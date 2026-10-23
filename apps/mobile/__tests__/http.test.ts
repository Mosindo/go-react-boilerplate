import { ApiError, mapErrorResponse } from "../src/api/errors";
import {
  buildQuery,
  createHttpClient,
  jwtExpiry,
  type TokenPair,
  type TokenStorage
} from "../src/api/http";
import { errorMessage } from "../src/i18n/errorMessage";
import { t } from "../src/i18n/fr";

function memoryStorage(initial: TokenPair | null): TokenStorage & { current: TokenPair | null } {
  const store = {
    current: initial,
    get: async () => store.current,
    set: async (tokens: TokenPair) => {
      store.current = tokens;
    },
    clear: async () => {
      store.current = null;
    }
  };
  return store;
}

function json(status: number, body: unknown, headers: Record<string, string> = {}) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", ...headers }
  });
}

const authResponse = (n: number) => ({
  accessToken: `access-${n}`,
  refreshToken: `refresh-${n}`,
  user: { id: "u1", email: "a@b.co", createdAt: "2024-01-01T00:00:00Z" }
});

describe("error mapping", () => {
  it("maps code, message and fields from the JSON body", () => {
    const error = mapErrorResponse(
      400,
      { error: "bad", code: "validation_failed", fields: { firstName: "required" } },
      null
    );
    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(400);
    expect(error.code).toBe("validation_failed");
    expect(error.fields).toEqual({ firstName: "required" });
  });

  it("falls back to a status based code and parses Retry-After", () => {
    const error = mapErrorResponse(429, null, "12");
    expect(error.code).toBe("rate_limited");
    expect(error.retryAfterSeconds).toBe(12);
    expect(errorMessage(error)).toBe(t.errors.rateLimitedIn(12));
  });

  it("shows French copy, never the backend text", () => {
    expect(errorMessage(mapErrorResponse(409, { error: "email taken", code: "email_taken" }))).toBe(
      t.errors.emailTaken
    );
    expect(errorMessage(mapErrorResponse(409, { error: "x", code: "photo_limit" }))).toBe(
      t.errors.photoLimit
    );
    expect(errorMessage(mapErrorResponse(500, { error: "boom" }))).toBe(t.errors.server);
    expect(errorMessage(new Error("english"))).toBe(t.errors.generic);
    expect(errorMessage(new ApiError(0, "network_error", "x"))).toBe(t.errors.network);
  });
});

describe("helpers", () => {
  it("builds query strings without empty values", () => {
    expect(buildQuery({ limit: 20, cursor: null, q: "a b" })).toBe("?limit=20&q=a%20b");
    expect(buildQuery(undefined)).toBe("");
  });

  it("reads the exp claim of a JWT", () => {
    const payload = Buffer.from(JSON.stringify({ exp: 1234 })).toString("base64url");
    expect(jwtExpiry(`h.${payload}.s`)).toBe(1234);
    expect(jwtExpiry("garbage")).toBeNull();
  });
});

describe("http client", () => {
  it("sends the bearer token and JSON body", async () => {
    const fetchImpl = jest.fn(async () => json(200, { ok: true }));
    const client = createHttpClient({
      baseUrl: "http://api",
      storage: memoryStorage({ accessToken: "a", refreshToken: "r" }),
      fetchImpl: fetchImpl as unknown as typeof fetch
    });
    await client.request("/x", { method: "POST", body: { a: 1 }, query: { limit: 5 } });
    const [url, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe("http://api/x?limit=5");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer a");
    expect(init.body).toBe('{"a":1}');
  });

  it("returns undefined on 204", async () => {
    const client = createHttpClient({
      baseUrl: "http://api",
      storage: memoryStorage({ accessToken: "a", refreshToken: "r" }),
      fetchImpl: (async () => new Response(null, { status: 204 })) as unknown as typeof fetch
    });
    await expect(client.request("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("refreshes once for concurrent 401s (single-flight) and retries each request once", async () => {
    const storage = memoryStorage({ accessToken: "old", refreshToken: "r0" });
    let refreshCalls = 0;
    const fetchImpl = jest.fn(async (url: string, init: RequestInit) => {
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        await new Promise((resolve) => setTimeout(resolve, 10));
        return json(200, authResponse(1));
      }
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer access-1"
        ? json(200, { path: url })
        : json(401, { code: "unauthorized", error: "expired" });
    });
    const refreshed = jest.fn();
    const client = createHttpClient({
      baseUrl: "http://api",
      storage,
      fetchImpl: fetchImpl as unknown as typeof fetch,
      onTokensRefreshed: refreshed
    });

    const results = await Promise.all([
      client.request("/a"),
      client.request("/b"),
      client.request("/c")
    ]);

    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
    expect(refreshed).toHaveBeenCalledTimes(1);
    expect(storage.current).toEqual({ accessToken: "access-1", refreshToken: "refresh-1" });
    // 3 first attempts + 1 refresh + 3 retries
    expect(fetchImpl).toHaveBeenCalledTimes(7);
  });

  it("does not retry more than once when the retried request is still 401", async () => {
    const storage = memoryStorage({ accessToken: "old", refreshToken: "r0" });
    const fetchImpl = jest.fn(async (url: string) =>
      url.endsWith("/auth/refresh")
        ? json(200, authResponse(1))
        : json(401, { code: "unauthorized", error: "no" })
    );
    const client = createHttpClient({
      baseUrl: "http://api",
      storage,
      fetchImpl: fetchImpl as unknown as typeof fetch
    });
    await expect(client.request("/a")).rejects.toMatchObject({ status: 401 });
    expect(fetchImpl).toHaveBeenCalledTimes(3);
  });

  it("signs out when the refresh token is rejected", async () => {
    const storage = memoryStorage({ accessToken: "old", refreshToken: "r0" });
    const expired = jest.fn();
    const fetchImpl = jest.fn(async (url: string) =>
      url.endsWith("/auth/refresh")
        ? json(401, { code: "unauthorized", error: "revoked" })
        : json(401, { error: "expired" })
    );
    const client = createHttpClient({
      baseUrl: "http://api",
      storage,
      fetchImpl: fetchImpl as unknown as typeof fetch,
      onSessionExpired: expired
    });
    await expect(client.request("/a")).rejects.toMatchObject({ status: 401 });
    expect(expired).toHaveBeenCalledTimes(1);
    expect(storage.current).toBeNull();
  });

  it("keeps the session when refresh fails because of the network", async () => {
    const storage = memoryStorage({ accessToken: "old", refreshToken: "r0" });
    const expired = jest.fn();
    const fetchImpl = jest.fn(async (url: string) => {
      if (url.endsWith("/auth/refresh")) throw new TypeError("offline");
      return json(401, { error: "expired" });
    });
    const client = createHttpClient({
      baseUrl: "http://api",
      storage,
      fetchImpl: fetchImpl as unknown as typeof fetch,
      onSessionExpired: expired
    });
    await expect(client.request("/a")).rejects.toMatchObject({ code: "network_error" });
    expect(expired).not.toHaveBeenCalled();
    expect(storage.current).not.toBeNull();
  });

  it("reuses tokens another request already rotated instead of refreshing again", async () => {
    const storage = memoryStorage({ accessToken: "old", refreshToken: "r0" });
    const fetchImpl = jest.fn(async (url: string, init: RequestInit) => {
      const auth = (init.headers as Record<string, string>).Authorization;
      if (auth === "Bearer fresh") return json(200, { ok: true });
      if (url.endsWith("/slow")) {
        storage.current = { accessToken: "fresh", refreshToken: "r1" };
        return json(401, { error: "expired" });
      }
      return json(500, {});
    });
    const client = createHttpClient({
      baseUrl: "http://api",
      storage,
      fetchImpl: fetchImpl as unknown as typeof fetch
    });
    await expect(client.request("/slow")).resolves.toEqual({ ok: true });
    expect(fetchImpl.mock.calls.some(([url]) => String(url).includes("/auth/refresh"))).toBe(false);
  });

  it("does not attach a token or refresh for public endpoints", async () => {
    const fetchImpl = jest.fn(async () =>
      json(401, { code: "unauthorized", error: "bad credentials" })
    );
    const client = createHttpClient({
      baseUrl: "http://api",
      storage: memoryStorage(null),
      fetchImpl: fetchImpl as unknown as typeof fetch
    });
    await expect(
      client.request("/auth/login", { method: "POST", body: {}, auth: false })
    ).rejects.toMatchObject({
      status: 401
    });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
  });

  it("maps transport failures to a network ApiError", async () => {
    const client = createHttpClient({
      baseUrl: "http://api",
      storage: memoryStorage({ accessToken: "a", refreshToken: "r" }),
      fetchImpl: (async () => {
        throw new TypeError("Network request failed");
      }) as unknown as typeof fetch
    });
    await expect(client.request("/x")).rejects.toMatchObject({ status: 0, code: "network_error" });
  });
});
