import { ApiError, apiRequest, setSessionLostHandler } from "../api/client";
import * as tokenStore from "../store/tokenStore";

jest.mock("../store/tokenStore", () => {
  let tokens: { accessToken: string; refreshToken: string } | null = { accessToken: "old-access", refreshToken: "old-refresh" };
  return {
    getTokens: jest.fn(async () => tokens),
    saveTokens: jest.fn(async (t: { accessToken: string; refreshToken: string }) => {
      tokens = t;
    }),
    clearTokens: jest.fn(async () => {
      tokens = null;
    })
  };
});

const json = (status: number, body?: unknown) =>
  Promise.resolve(
    new Response(body === undefined ? null : JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } })
  );

describe("apiRequest", () => {
  const originalFetch = global.fetch;
  afterEach(() => {
    global.fetch = originalFetch;
    setSessionLostHandler(null);
  });

  it("sends the bearer token and parses JSON", async () => {
    const fetchMock = jest.fn(() => json(200, { ok: true }));
    global.fetch = fetchMock as unknown as typeof fetch;
    await expect(apiRequest<{ ok: boolean }>("/me")).resolves.toEqual({ ok: true });
    const init = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1];
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer old-access");
  });

  it("refreshes once on 401 (shared by concurrent calls) and retries", async () => {
    let refreshCalls = 0;
    const fetchMock = jest.fn((url: string, init?: RequestInit) => {
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return json(200, { accessToken: "new-access", refreshToken: "new-refresh" });
      }
      const auth = (init?.headers as Record<string, string>).Authorization;
      return auth === "Bearer new-access" ? json(200, { n: 1 }) : json(401, { error: "invalid token" });
    });
    global.fetch = fetchMock as unknown as typeof fetch;

    const results = await Promise.all([apiRequest("/a"), apiRequest("/b"), apiRequest("/c")]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
    expect(tokenStore.saveTokens).toHaveBeenCalledWith({ accessToken: "new-access", refreshToken: "new-refresh" });
  });

  it("signals a lost session when the refresh fails", async () => {
    const lost = jest.fn();
    setSessionLostHandler(lost);
    global.fetch = jest.fn(() => json(401, { error: "nope" })) as unknown as typeof fetch;
    await expect(apiRequest("/me")).rejects.toBeInstanceOf(ApiError);
    expect(lost).toHaveBeenCalled();
  });

  it("surfaces the server error message", async () => {
    global.fetch = jest.fn(() => json(422, { error: "gender is invalid" })) as unknown as typeof fetch;
    await expect(apiRequest("/x", { method: "PUT", body: {} })).rejects.toMatchObject({ status: 422, message: "gender is invalid" });
  });

  it("maps network failures to a readable error", async () => {
    global.fetch = jest.fn(() => Promise.reject(new TypeError("Network request failed"))) as unknown as typeof fetch;
    await expect(apiRequest("/x")).rejects.toMatchObject({ status: 0 });
  });

  it("returns undefined for 204 responses", async () => {
    global.fetch = jest.fn(() => Promise.resolve(new Response(null, { status: 204 }))) as unknown as typeof fetch;
    await expect(apiRequest("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });
});
