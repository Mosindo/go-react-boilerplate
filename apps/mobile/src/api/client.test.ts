import { describe, expect, it, vi } from "vitest";
import { TokenStore, type TokenBackend } from "../store/tokenStore";
import { ApiClient, ApiError, NetworkError } from "./client";

function memoryBackend(initial?: string): TokenBackend & { value: string | null } {
  const backend = {
    value: initial ?? null,
    get: async () => backend.value,
    set: async (v: string) => void (backend.value = v),
    remove: async () => void (backend.value = null)
  };
  return backend;
}

const json = (status: number, body: unknown, headers: Record<string, string> = {}) =>
  new Response(body === undefined ? null : JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } });

async function setup(handler: (url: string, init: RequestInit) => Response | Promise<Response>) {
  const backend = memoryBackend(JSON.stringify({ accessToken: "access-1", refreshToken: "refresh-1" }));
  const tokens = new TokenStore(backend);
  const calls: { url: string; init: RequestInit }[] = [];
  const fetchImpl = vi.fn(async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return handler(url, init);
  });
  const expired = vi.fn();
  const client = new ApiClient({
    baseUrl: "http://api.test",
    tokens,
    fetchImpl: fetchImpl as unknown as typeof fetch,
    onSessionExpired: expired
  });
  return { client, calls, expired, backend, tokens };
}

const authHeader = (init: RequestInit) => (init.headers as Record<string, string>).Authorization;

describe("ApiClient", () => {
  it("sends the bearer token and parses JSON", async () => {
    const { client, calls } = await setup(() => json(200, { ok: true }));
    expect(await client.request("/me")).toEqual({ ok: true });
    expect(calls[0].url).toBe("http://api.test/me");
    expect(authHeader(calls[0].init)).toBe("Bearer access-1");
  });

  it("does not send a token on public calls", async () => {
    const { client, calls } = await setup(() => json(200, {}));
    await client.request("/auth/login", { method: "POST", body: { a: 1 }, auth: false });
    expect(authHeader(calls[0].init)).toBeUndefined();
    expect((calls[0].init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    expect(calls[0].init.body).toBe('{"a":1}');
  });

  it("maps API errors to ApiError with the stable code", async () => {
    const { client } = await setup(() => json(409, { error: "already reacted", code: "already_swiped" }));
    await expect(client.request("/swipes", { method: "POST", body: {} })).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      code: "already_swiped",
      message: "already reacted"
    });
  });

  it("exposes Retry-After on rate limiting", async () => {
    const { client } = await setup(() => json(429, { error: "too many requests", code: "rate_limited" }, { "Retry-After": "12" }));
    await expect(client.request("/x")).rejects.toMatchObject({ status: 429, retryAfterSeconds: 12 });
  });

  it("copes with non-JSON error bodies", async () => {
    const { client } = await setup(() => new Response("<html>bad gateway</html>", { status: 502 }));
    await expect(client.request("/x")).rejects.toMatchObject({ status: 502, code: "http_error" });
  });

  it("returns undefined for empty successful responses", async () => {
    const { client } = await setup(() => new Response(null, { status: 204 }));
    expect(await client.request("/x", { method: "DELETE" })).toBeUndefined();
  });

  it("refreshes the access token once on 401 and replays the request", async () => {
    let first = true;
    const { client, calls, backend } = await setup((url) => {
      if (url.endsWith("/auth/refresh")) return json(200, { accessToken: "access-2", refreshToken: "refresh-2" });
      if (first) {
        first = false;
        return json(401, { error: "invalid token" });
      }
      return json(200, { ok: 1 });
    });
    expect(await client.request("/me")).toEqual({ ok: 1 });
    expect(calls.map((c) => c.url.replace("http://api.test", ""))).toEqual(["/me", "/auth/refresh", "/me"]);
    expect(authHeader(calls[2].init)).toBe("Bearer access-2");
    expect(JSON.parse(backend.value!)).toEqual({ accessToken: "access-2", refreshToken: "refresh-2" });
  });

  it("shares one refresh between concurrent 401s (a rotated refresh token must not be reused)", async () => {
    let refreshes = 0;
    const { client } = await setup(async (url, init) => {
      if (url.endsWith("/auth/refresh")) {
        refreshes++;
        await new Promise((r) => setTimeout(r, 10));
        return json(200, { accessToken: "access-2", refreshToken: "refresh-2" });
      }
      return authHeader(init) === "Bearer access-2" ? json(200, { ok: 1 }) : json(401, { error: "expired" });
    });
    const results = await Promise.all([client.request("/a"), client.request("/b"), client.request("/c")]);
    expect(results).toHaveLength(3);
    expect(refreshes).toBe(1);
  });

  it("ends the session when the refresh token is rejected", async () => {
    const { client, expired, backend } = await setup((url) =>
      url.endsWith("/auth/refresh") ? json(401, { error: "invalid refresh token" }) : json(401, { error: "expired" })
    );
    await expect(client.request("/me")).rejects.toMatchObject({ status: 401, code: "unauthorized" });
    expect(expired).toHaveBeenCalledOnce();
    expect(backend.value).toBeNull();
  });

  it("keeps the session when the refresh fails because the server is down", async () => {
    const { client, expired, backend } = await setup((url) =>
      url.endsWith("/auth/refresh") ? json(503, { error: "down" }) : json(401, { error: "expired" })
    );
    await expect(client.request("/me")).rejects.toBeInstanceOf(NetworkError);
    expect(expired).not.toHaveBeenCalled();
    expect(backend.value).not.toBeNull();
  });

  it("keeps the session when offline", async () => {
    const backend = memoryBackend(JSON.stringify({ accessToken: "a", refreshToken: "r" }));
    const expired = vi.fn();
    const client = new ApiClient({
      baseUrl: "http://api.test",
      tokens: new TokenStore(backend),
      onSessionExpired: expired,
      fetchImpl: (async () => Promise.reject(new TypeError("Failed to fetch"))) as typeof fetch
    });
    await expect(client.request("/me")).rejects.toBeInstanceOf(NetworkError);
    expect(expired).not.toHaveBeenCalled();
  });

  it("fails fast with 401 when there is no session", async () => {
    const tokens = new TokenStore(memoryBackend());
    const fetchImpl = vi.fn();
    const client = new ApiClient({ baseUrl: "http://api.test", tokens, fetchImpl: fetchImpl as unknown as typeof fetch });
    await expect(client.request("/me")).rejects.toBeInstanceOf(ApiError);
    expect(fetchImpl).not.toHaveBeenCalled();
  });

  it("builds absolute media URLs", async () => {
    const { client } = await setup(() => json(200, {}));
    expect(client.mediaUrl("/photos/1/image?sig=x")).toBe("http://api.test/photos/1/image?sig=x");
    expect(client.mediaUrl("https://cdn.test/a.jpg")).toBe("https://cdn.test/a.jpg");
  });

  it("signs the user out when the realtime layer's refresh is rejected", async () => {
    const { client, expired } = await setup(() => json(401, { error: "invalid refresh token" }));
    expect(await client.ensureFreshAccessToken()).toBeNull();
    expect(expired).toHaveBeenCalledOnce();
  });
});
