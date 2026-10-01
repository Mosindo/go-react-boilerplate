import { ApiError, apiRequest, setAuthBridge, type AuthBridge } from "../client";

jest.mock("../../shared/feedback/store", () => ({ showGlobalError: jest.fn() }));

process.env.EXPO_PUBLIC_API_URL = "http://api.test";

type FetchMock = jest.Mock<Promise<Response>, [string, RequestInit]>;

function json(status: number, body?: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function setup(responses: Response[]): { fetchMock: FetchMock; bridge: AuthBridge } {
  const fetchMock: FetchMock = jest.fn();
  responses.forEach((r) => fetchMock.mockResolvedValueOnce(r));
  global.fetch = fetchMock as unknown as typeof fetch;
  let tokens = { accessToken: "old", refreshToken: "refresh-1" };
  const bridge: AuthBridge = {
    getTokens: jest.fn(async () => tokens),
    setTokens: jest.fn(async (next) => {
      tokens = next;
    }),
    onAuthLost: jest.fn()
  };
  setAuthBridge(bridge);
  return { fetchMock, bridge };
}

afterEach(() => setAuthBridge(null));

describe("apiRequest", () => {
  it("sends the bearer token and parses JSON", async () => {
    const { fetchMock } = setup([json(200, { ok: true })]);
    await expect(apiRequest<{ ok: boolean }>("/me")).resolves.toEqual({ ok: true });
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://api.test/me");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer old");
  });

  it("serializes bodies and query strings", async () => {
    const { fetchMock } = setup([json(200, {})]);
    await apiRequest("/swipes", { method: "POST", body: { userId: "u", action: "like" }, query: { limit: 5, skip: undefined } });
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://api.test/swipes?limit=5");
    expect(init.body).toBe(JSON.stringify({ userId: "u", action: "like" }));
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("returns undefined for 204", async () => {
    setup([new Response(null, { status: 204 })]);
    await expect(apiRequest("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("does not attach tokens to public endpoints", async () => {
    const { fetchMock } = setup([json(200, {})]);
    await apiRequest("/auth/login", { method: "POST", body: {}, auth: false });
    expect((fetchMock.mock.calls[0]?.[1].headers as Record<string, string>).Authorization).toBeUndefined();
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    const { fetchMock, bridge } = setup([
      json(401, { error: "invalid token" }),
      json(200, { accessToken: "new", refreshToken: "refresh-2", user: { id: "u" } }),
      json(200, { me: true })
    ]);
    await expect(apiRequest("/me")).resolves.toEqual({ me: true });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1]?.[0]).toBe("http://api.test/auth/refresh");
    expect((fetchMock.mock.calls[2]?.[1].headers as Record<string, string>).Authorization).toBe("Bearer new");
    expect(bridge.setTokens).toHaveBeenCalledWith({ accessToken: "new", refreshToken: "refresh-2" }, { id: "u" });
  });

  it("shares a single refresh between concurrent 401s", async () => {
    const { fetchMock } = setup([
      json(401, { error: "x" }),
      json(401, { error: "x" }),
      json(200, { accessToken: "new", refreshToken: "r2", user: { id: "u" } }),
      json(200, { n: 1 }),
      json(200, { n: 2 })
    ]);
    const results = await Promise.all([apiRequest("/a"), apiRequest("/b")]);
    expect(results).toEqual([{ n: 1 }, { n: 2 }]);
    const refreshCalls = fetchMock.mock.calls.filter(([url]) => url.endsWith("/auth/refresh"));
    expect(refreshCalls).toHaveLength(1);
  });

  it("ends the session when the refresh token is rejected", async () => {
    const { bridge } = setup([json(401, { error: "x" }), json(401, { error: "invalid refresh token" })]);
    await expect(apiRequest("/me")).rejects.toMatchObject({ status: 401 });
    expect(bridge.onAuthLost).toHaveBeenCalledTimes(1);
  });

  it("surfaces the API error message for 4xx", async () => {
    setup([json(400, { error: "you must be at least 18 years old" })]);
    const error = await apiRequest("/me/profile", { method: "PUT", body: {} }).catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).message).toBe("you must be at least 18 years old");
    expect((error as ApiError).status).toBe(400);
  });

  it("hides server internals behind a friendly message for 5xx", async () => {
    setup([json(500, { error: "pq: relation does not exist" })]);
    const error = (await apiRequest("/x").catch((e: unknown) => e)) as ApiError;
    expect(error.status).toBe(500);
    expect(error.message).not.toMatch(/pq:/);
  });

  it("maps network failures to a status 0 ApiError", async () => {
    setup([]);
    (global.fetch as unknown as jest.Mock).mockRejectedValueOnce(new TypeError("Network request failed"));
    await expect(apiRequest("/x")).rejects.toMatchObject({ status: 0 });
  });
});
