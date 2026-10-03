import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../store/tokenStore", () => ({ getAccessToken: vi.fn(async () => "token-1") }));
vi.mock("../shared/feedback", () => ({ showGlobalError: vi.fn() }));

const jsonResponse = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), { status, headers: { "content-type": "application/json" } });

describe("apiRequest", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.resetModules();
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    process.env.EXPO_PUBLIC_API_URL = "http://api.test";
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends the bearer token and parses JSON", async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { ok: true }));
    const { apiRequest } = await import("./client");
    await expect(apiRequest("/ping")).resolves.toEqual({ ok: true });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://api.test/ping");
    expect((init.headers as Headers).get("Authorization")).toBe("Bearer token-1");
  });

  it("surfaces server errors with their code", async () => {
    fetchMock.mockResolvedValue(jsonResponse(409, { error: "complete your profile", code: "profile_incomplete" }));
    const { apiRequest, ApiError } = await import("./client");
    const error = await apiRequest("/discover").catch((e) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 409, code: "profile_incomplete", message: "complete your profile" });
  });

  it("refreshes the session once on 401 and retries", async () => {
    const { setRefreshHandler } = await import("./session");
    const refresh = vi.fn(async () => true);
    setRefreshHandler(refresh);
    fetchMock.mockResolvedValueOnce(jsonResponse(401, { error: "invalid token" })).mockResolvedValueOnce(jsonResponse(200, { done: 1 }));
    const { apiRequest } = await import("./client");
    await expect(apiRequest("/me/profile")).resolves.toEqual({ done: 1 });
    expect(refresh).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("shares one refresh between concurrent 401s", async () => {
    const { setRefreshHandler } = await import("./session");
    let resolveRefresh: (v: boolean) => void = () => undefined;
    const refresh = vi.fn(() => new Promise<boolean>((r) => (resolveRefresh = r)));
    setRefreshHandler(refresh);
    fetchMock.mockImplementation(async () => (fetchMock.mock.calls.length <= 2 ? jsonResponse(401, {}) : jsonResponse(200, { ok: 1 })));
    const { apiRequest } = await import("./client");
    const calls = Promise.all([apiRequest("/a"), apiRequest("/b")]);
    await vi.waitFor(() => expect(refresh).toHaveBeenCalled());
    resolveRefresh(true);
    await calls;
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("does not loop when the refresh fails", async () => {
    const { setRefreshHandler } = await import("./session");
    setRefreshHandler(async () => false);
    fetchMock.mockResolvedValue(jsonResponse(401, { error: "invalid token" }));
    const { apiRequest } = await import("./client");
    await expect(apiRequest("/me")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("never tries to refresh auth endpoints themselves", async () => {
    const { setRefreshHandler } = await import("./session");
    const refresh = vi.fn(async () => true);
    setRefreshHandler(refresh);
    fetchMock.mockResolvedValue(jsonResponse(401, { error: "invalid credentials" }));
    const { apiRequest } = await import("./client");
    await expect(apiRequest("/auth/login", { method: "POST" })).rejects.toMatchObject({ status: 401 });
    expect(refresh).not.toHaveBeenCalled();
  });
});
