import { ApiError, apiRequest, buildQuery, refreshAccessToken, setSessionExpiredListener } from "../src/api/client";
import { resolveApiBaseUrl, toWebSocketUrl } from "../src/api/config";
import * as tokenStore from "../src/store/tokenStore";

jest.mock("../src/store/tokenStore", () => {
  let tokens: { accessToken: string; refreshToken: string } | null = { accessToken: "old", refreshToken: "r1" };
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

type FetchMock = jest.Mock<Promise<Response>, [string, RequestInit?]>;
const json = (status: number, body?: unknown): Response =>
  ({
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
    json: async () => body
  }) as Response;

describe("api config", () => {
  it("prefers the explicit URL and strips trailing slashes", () => {
    expect(resolveApiBaseUrl("https://api.x.com//", "ios", true)).toBe("https://api.x.com");
  });
  it("falls back per platform in dev only", () => {
    expect(resolveApiBaseUrl(undefined, "android", true)).toBe("http://10.0.2.2:18080");
    expect(resolveApiBaseUrl(undefined, "ios", true)).toBe("http://localhost:18080");
    expect(resolveApiBaseUrl(undefined, "ios", false)).toBe("");
  });
  it("derives the websocket URL", () => {
    expect(toWebSocketUrl("http://a:1")).toBe("ws://a:1/ws");
    expect(toWebSocketUrl("https://a")).toBe("wss://a/ws");
  });
});

describe("buildQuery", () => {
  it("skips empty values and encodes", () => {
    expect(buildQuery({ limit: 10, cursor: null, q: "a b" })).toBe("?limit=10&q=a%20b");
    expect(buildQuery({})).toBe("");
  });
});

describe("apiRequest", () => {
  let fetchMock: FetchMock;
  beforeEach(() => {
    fetchMock = jest.fn();
    global.fetch = fetchMock as unknown as typeof fetch;
    jest.mocked(tokenStore.getTokens).mockClear();
  });

  it("maps error payloads to ApiError with code", async () => {
    fetchMock.mockResolvedValueOnce(json(409, { error: "taken", code: "email_taken" }));
    await expect(apiRequest("/x", { auth: false })).rejects.toMatchObject({ name: "ApiError", status: 409, code: "email_taken", message: "taken" });
  });

  it("returns undefined on 204", async () => {
    fetchMock.mockResolvedValueOnce(json(204));
    await expect(apiRequest("/x")).resolves.toBeUndefined();
  });

  it("maps network failures", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("fail"));
    await expect(apiRequest("/x")).rejects.toMatchObject({ code: "network" });
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "expired" }))
      .mockResolvedValueOnce(json(200, { accessToken: "new", refreshToken: "r2", user: {} }))
      .mockResolvedValueOnce(json(200, { ok: true }));
    await expect(apiRequest("/me")).resolves.toEqual({ ok: true });
    const retry = fetchMock.mock.calls[2];
    expect((retry?.[1]?.headers as Record<string, string>).Authorization).toBe("Bearer new");
  });

  it("shares a single refresh between concurrent 401s", async () => {
    await tokenStore.saveTokens({ accessToken: "old", refreshToken: "r2" });
    let refreshCalls = 0;
    fetchMock.mockImplementation(async (url, init) => {
      if (url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return json(200, { accessToken: "fresh", refreshToken: "r3", user: {} });
      }
      const auth = (init?.headers as Record<string, string>).Authorization;
      return auth === "Bearer fresh" ? json(200, { ok: true }) : json(401, { error: "expired" });
    });
    const results = await Promise.all([apiRequest("/a"), apiRequest("/b"), apiRequest("/c")]);
    expect(results).toEqual([{ ok: true }, { ok: true }, { ok: true }]);
    expect(refreshCalls).toBe(1);
  });

  it("signals session expiry when the refresh token is rejected", async () => {
    await tokenStore.saveTokens({ accessToken: "old", refreshToken: "dead" });
    const expired = jest.fn();
    setSessionExpiredListener(expired);
    fetchMock.mockResolvedValueOnce(json(401, { error: "expired" })).mockResolvedValueOnce(json(401, { error: "invalid refresh token" }));
    await expect(apiRequest("/me")).rejects.toBeInstanceOf(ApiError);
    expect(expired).toHaveBeenCalledTimes(1);
    expect(tokenStore.clearTokens).toHaveBeenCalled();
    setSessionExpiredListener(null);
  });

  it("refreshAccessToken resolves null without a stored refresh token", async () => {
    await expect(refreshAccessToken()).resolves.toBeNull();
  });
});
