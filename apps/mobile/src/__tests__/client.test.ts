import { ApiError, onSessionExpired, request } from "../api/client";
import { clearTokens, currentTokens, saveTokens } from "../api/tokenStore";

type MockResponse = { status: number; body?: unknown };

function mockFetch(handler: (url: string, init: RequestInit) => MockResponse) {
  const calls: { url: string; init: RequestInit }[] = [];
  global.fetch = jest.fn(async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const { status, body } = handler(url, init);
    return {
      ok: status >= 200 && status < 300,
      status,
      json: async () => body,
      text: async () => (body === undefined ? "" : JSON.stringify(body))
    } as unknown as Response;
  }) as unknown as typeof fetch;
  return calls;
}

beforeEach(async () => {
  await clearTokens();
});

describe("api client", () => {
  it("sends the bearer token and parses JSON", async () => {
    await saveTokens({ accessToken: "a1", refreshToken: "r1" });
    const calls = mockFetch(() => ({ status: 200, body: { ok: true } }));
    await expect(request<{ ok: boolean }>("/ping")).resolves.toEqual({ ok: true });
    expect((calls[0].init.headers as Record<string, string>).Authorization).toBe("Bearer a1");
  });

  it("maps API errors to ApiError with the server message", async () => {
    mockFetch(() => ({ status: 409, body: { error: "photo limit reached" } }));
    await expect(request("/photos", { method: "POST" })).rejects.toMatchObject({
      name: "ApiError",
      status: 409,
      message: "photo limit reached"
    });
  });

  it("returns undefined on 204", async () => {
    mockFetch(() => ({ status: 204 }));
    await expect(request("/blocks/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("refreshes once on 401 (single flight) and retries the request", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "r1" });
    let refreshes = 0;
    const calls = mockFetch((url, init) => {
      if (url.endsWith("/auth/refresh")) {
        refreshes += 1;
        return { status: 200, body: { accessToken: "new", refreshToken: "r2", user: { id: "u" } } };
      }
      const auth = (init.headers as Record<string, string>).Authorization;
      return auth === "Bearer new"
        ? { status: 200, body: { done: true } }
        : { status: 401, body: { error: "invalid token" } };
    });

    const results = await Promise.all([request("/a"), request("/b")]);
    expect(results).toEqual([{ done: true }, { done: true }]);
    expect(refreshes).toBe(1);
    expect(currentTokens()).toEqual({ accessToken: "new", refreshToken: "r2" });
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
  });

  it("signs out when the refresh token is rejected", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "dead" });
    mockFetch(() => ({ status: 401, body: { error: "invalid refresh token" } }));
    const expired = jest.fn();
    const off = onSessionExpired(expired);
    await expect(request("/me")).rejects.toBeInstanceOf(ApiError);
    off();
    expect(expired).toHaveBeenCalledTimes(1);
    expect(currentTokens()).toBeNull();
  });

  it("does not try to refresh anonymous requests", async () => {
    const calls = mockFetch(() => ({ status: 401, body: { error: "invalid credentials" } }));
    await expect(
      request("/auth/login", { method: "POST", body: {}, anonymous: true })
    ).rejects.toMatchObject({ status: 401 });
    expect(calls).toHaveLength(1);
  });

  it("reports network failures as status 0", async () => {
    global.fetch = jest.fn(async () => {
      throw new TypeError("Network request failed");
    }) as unknown as typeof fetch;
    await expect(request("/me")).rejects.toMatchObject({ status: 0 });
  });
});
