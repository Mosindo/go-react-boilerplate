import { ApiClient, ApiError, getErrorMessage, mapErrorResponse } from "../client";
import { TokenStore, type StorageAdapter } from "../../store/tokenStore";

function memoryStorage(): StorageAdapter {
  const data = new Map<string, string>();
  return {
    get: async (k) => data.get(k) ?? null,
    set: async (k, v) => void data.set(k, v),
    remove: async (k) => void data.delete(k)
  };
}

function json(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json", ...headers }
  });
}

async function setup(fetchImpl: jest.Mock) {
  const tokens = new TokenStore(memoryStorage());
  await tokens.set({ accessToken: "old-access", refreshToken: "old-refresh" });
  const onAuthLost = jest.fn();
  const client = new ApiClient({
    baseUrl: "http://api.test",
    tokens,
    fetchImpl: fetchImpl as unknown as typeof fetch,
    onAuthLost
  });
  return { client, tokens, onAuthLost };
}

const authResponse = (n: number) => ({
  accessToken: `access-${n}`,
  refreshToken: `refresh-${n}`,
  user: { id: "u1", email: "a@b.co", profileCompleted: true, createdAt: "2026-01-01T00:00:00Z" }
});

describe("ApiClient", () => {
  it("sends the bearer token and parses JSON", async () => {
    const fetchImpl = jest.fn(async () => json(200, { ok: true }));
    const { client } = await setup(fetchImpl);
    await expect(client.get("/me", { query: { limit: 5, cursor: undefined } })).resolves.toEqual({ ok: true });
    const [url, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe("http://api.test/me?limit=5");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer old-access");
  });

  it("returns undefined for 204", async () => {
    const { client } = await setup(jest.fn(async () => new Response(null, { status: 204 })));
    await expect(client.post("/x")).resolves.toBeUndefined();
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    const fetchImpl = jest.fn(async (url: string, init: RequestInit) => {
      if (url.endsWith("/auth/refresh")) return json(200, authResponse(1));
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer access-1"
        ? json(200, { done: true })
        : json(401, { error: "expired", code: "unauthorized" });
    });
    const { client, tokens } = await setup(fetchImpl);
    await expect(client.get("/profile")).resolves.toEqual({ done: true });
    expect(fetchImpl).toHaveBeenCalledTimes(3);
    expect(tokens.getTokens()).toEqual({ accessToken: "access-1", refreshToken: "refresh-1" });
  });

  it("single-flights concurrent refreshes", async () => {
    let refreshCalls = 0;
    const fetchImpl = jest.fn(async (url: string, init: RequestInit) => {
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        await new Promise((r) => setTimeout(r, 20));
        return json(200, authResponse(1));
      }
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer access-1" ? json(200, { ok: 1 }) : json(401, { code: "unauthorized", error: "no" });
    });
    const { client } = await setup(fetchImpl);
    const results = await Promise.all([client.get("/a"), client.get("/b"), client.get("/c")]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
  });

  it("does not loop: a second 401 after refresh is surfaced", async () => {
    const fetchImpl = jest.fn(async (url: string) =>
      url.endsWith("/auth/refresh") ? json(200, authResponse(1)) : json(401, { code: "unauthorized", error: "nope" })
    );
    const { client } = await setup(fetchImpl);
    await expect(client.get("/profile")).rejects.toMatchObject({ status: 401, code: "unauthorized" });
    expect(fetchImpl).toHaveBeenCalledTimes(3);
  });

  it("clears the session and notifies when the refresh token is rejected", async () => {
    const fetchImpl = jest.fn(async (url: string) =>
      url.endsWith("/auth/refresh")
        ? json(401, { code: "unauthorized", error: "revoked" })
        : json(401, { code: "unauthorized", error: "expired" })
    );
    const { client, tokens, onAuthLost } = await setup(fetchImpl);
    await expect(client.get("/me")).rejects.toBeInstanceOf(ApiError);
    expect(tokens.getTokens()).toBeNull();
    expect(onAuthLost).toHaveBeenCalledTimes(1);
  });

  it("keeps the session when the refresh fails because of the network", async () => {
    const fetchImpl = jest.fn(async (url: string) => {
      if (url.endsWith("/auth/refresh")) throw new TypeError("offline");
      return json(401, { code: "unauthorized", error: "expired" });
    });
    const { client, tokens, onAuthLost } = await setup(fetchImpl);
    await expect(client.get("/me")).rejects.toMatchObject({ code: "network" });
    expect(tokens.getTokens()).not.toBeNull();
    expect(onAuthLost).not.toHaveBeenCalled();
  });

  it("does not try to refresh for unauthenticated calls (wrong password)", async () => {
    const fetchImpl = jest.fn(async () => json(401, { code: "unauthorized", error: "invalid credentials" }));
    const { client } = await setup(fetchImpl);
    await expect(client.post("/auth/login", { email: "a", password: "b" }, { auth: false })).rejects.toMatchObject({
      status: 401
    });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it("maps network failures", async () => {
    const { client } = await setup(jest.fn(async () => Promise.reject(new TypeError("Failed to fetch"))));
    await expect(client.get("/x")).rejects.toMatchObject({ status: 0, code: "network" });
  });
});

describe("mapErrorResponse", () => {
  it("reads code, message and field errors", () => {
    const err = mapErrorResponse(400, {
      error: "bad",
      code: "validation_failed",
      fields: { displayName: "too short", n: 3 }
    });
    expect(err).toMatchObject({
      status: 400,
      code: "validation_failed",
      message: "bad",
      fields: { displayName: "too short" }
    });
  });

  it("falls back to the status when the body has no code", () => {
    expect(mapErrorResponse(404, null).code).toBe("not_found");
    expect(mapErrorResponse(503, "oops").code).toBe("internal");
    expect(mapErrorResponse(418, {}).code).toBe("unknown");
  });

  it("keeps profile_incomplete and parses Retry-After", () => {
    expect(mapErrorResponse(409, { code: "profile_incomplete", error: "x" }).code).toBe("profile_incomplete");
    const limited = mapErrorResponse(429, { code: "rate_limited", error: "slow down" }, "12");
    expect(limited.retryAfter).toBe(12);
    expect(getErrorMessage(limited)).toContain("12s");
  });
});
