import { ApiError, request, setSessionListeners, _refreshInFlight } from "../api/client";
import { loadTokens, resetTokenCacheForTests, saveTokens } from "../lib/tokenStore";

type Handler = (url: string, init: RequestInit) => Response | Promise<Response>;

function mockFetch(handler: Handler) {
  const calls: { url: string; init: RequestInit }[] = [];
  globalThis.fetch = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init: init ?? {} });
    return handler(String(url), init ?? {});
  }) as unknown as typeof fetch;
  return calls;
}

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

beforeEach(async () => {
  resetTokenCacheForTests();
  await saveTokens({ accessToken: "old-access", refreshToken: "old-refresh" });
  setSessionListeners({ onExpired: () => undefined });
});

describe("api client", () => {
  it("sends the bearer token and parses JSON", async () => {
    const calls = mockFetch(() => json(200, { ok: true }));
    await expect(request<{ ok: boolean }>("/me")).resolves.toEqual({ ok: true });
    expect((calls[0]?.init.headers as Record<string, string>).Authorization).toBe("Bearer old-access");
  });

  it("maps API errors to ApiError with the server message and code", async () => {
    mockFetch(() => json(409, { error: "déjà fait", code: "already_swiped" }));
    const err = await request("/x", { method: "POST", body: {} }).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 409, code: "already_swiped", message: "déjà fait" });
  });

  it("reports network failures as status 0", async () => {
    globalThis.fetch = jest.fn(async () => {
      throw new TypeError("offline");
    }) as unknown as typeof fetch;
    const err = (await request("/x").catch((e: unknown) => e)) as ApiError;
    expect(err.isNetwork).toBe(true);
  });

  it("returns undefined on 204", async () => {
    mockFetch(() => new Response(null, { status: 204 }));
    await expect(request("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("refreshes once on 401, retries, and shares one refresh between concurrent requests", async () => {
    let refreshCalls = 0;
    const calls = mockFetch((url, init) => {
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return json(200, { accessToken: "new-access", refreshToken: "new-refresh" });
      }
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer new-access" ? json(200, { ok: true }) : json(401, { error: "expired" });
    });
    const results = await Promise.all([request("/a"), request("/b"), request("/c")]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
    expect(await loadTokens()).toEqual({ accessToken: "new-access", refreshToken: "new-refresh" });
    expect(_refreshInFlight()).toBeNull();
  });

  it("logs out when the refresh token is rejected", async () => {
    const onExpired = jest.fn();
    setSessionListeners({ onExpired });
    mockFetch((url) => (url.endsWith("/auth/refresh") ? json(401, { error: "bad" }) : json(401, { error: "expired" })));
    await expect(request("/me")).rejects.toMatchObject({ status: 401 });
    expect(onExpired).toHaveBeenCalledTimes(1);
    expect(await loadTokens()).toBeNull();
  });

  it("keeps the session when refresh fails because the network is down", async () => {
    mockFetch((url) => {
      if (url.endsWith("/auth/refresh")) throw new TypeError("offline");
      return json(401, { error: "expired" });
    });
    await expect(request("/me")).rejects.toMatchObject({ status: 401 });
    expect(await loadTokens()).toEqual({ accessToken: "old-access", refreshToken: "old-refresh" });
  });

  it("does not attach credentials to unauthenticated calls", async () => {
    const calls = mockFetch(() => json(200, {}));
    await request("/auth/login", { method: "POST", body: { email: "a", password: "b" }, auth: false });
    expect((calls[0]?.init.headers as Record<string, string>).Authorization).toBeUndefined();
  });
});
