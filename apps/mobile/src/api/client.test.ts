import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("react-native", () => ({ Platform: { OS: "ios" } }));

import { ApiError, api, configureSession, readExpiry, setAccessToken } from "./client";

function jwt(expSeconds: number): string {
  const enc = (o: object) => Buffer.from(JSON.stringify(o)).toString("base64url");
  return `${enc({ alg: "HS256" })}.${enc({ exp: expSeconds })}.sig`;
}

const json = (status: number, body?: unknown) =>
  new Response(body === undefined ? "" : JSON.stringify(body), { status });

describe("api client", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
    fetchMock.mockReset();
    setAccessToken("expired");
    configureSession(null);
  });
  afterEach(() => vi.unstubAllGlobals());

  it("reads the expiry of a JWT and tolerates garbage", () => {
    expect(readExpiry(jwt(1_900_000_000))).toBe(1_900_000_000_000);
    expect(readExpiry("not-a-jwt")).toBe(0);
  });

  it("refreshes once on 401 and replays the request with the new token", async () => {
    const onTokens = vi.fn();
    configureSession({ getRefreshToken: async () => "r1", onTokens, onExpired: vi.fn() });
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "invalid token" }))
      .mockResolvedValueOnce(
        json(200, { accessToken: "fresh", refreshToken: "r2", user: { id: "u" } }),
      )
      .mockResolvedValueOnce(json(200, { ok: true }));

    await expect(api.get("/me")).resolves.toEqual({ ok: true });
    expect(onTokens).toHaveBeenCalledOnce();
    const replay = fetchMock.mock.calls[2]?.[1]?.headers as Record<string, string>;
    expect(replay.Authorization).toBe("Bearer fresh");
  });

  it("shares a single refresh between concurrent 401s", async () => {
    configureSession({ getRefreshToken: async () => "r1", onTokens: vi.fn(), onExpired: vi.fn() });
    let refreshCalls = 0;
    fetchMock.mockImplementation(async (url, init) => {
      if (String(url).endsWith("/auth/refresh")) {
        refreshCalls++;
        return json(200, { accessToken: "fresh", refreshToken: "r2", user: { id: "u" } });
      }
      const auth = (init?.headers as Record<string, string>).Authorization;
      return auth === "Bearer fresh"
        ? json(200, { ok: true })
        : json(401, { error: "invalid token" });
    });
    await Promise.all([api.get("/a"), api.get("/b"), api.get("/c")]);
    expect(refreshCalls).toBe(1);
  });

  it("ends the session when the refresh token is rejected", async () => {
    const onExpired = vi.fn();
    configureSession({ getRefreshToken: async () => "r1", onTokens: vi.fn(), onExpired });
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "invalid token" }))
      .mockResolvedValueOnce(json(401, { error: "invalid refresh token" }));
    await expect(api.get("/me")).rejects.toMatchObject({ status: 401 });
    expect(onExpired).toHaveBeenCalledOnce();
  });

  it("does not log out on a transient refresh failure", async () => {
    const onExpired = vi.fn();
    configureSession({ getRefreshToken: async () => "r1", onTokens: vi.fn(), onExpired });
    fetchMock.mockResolvedValueOnce(json(401, {})).mockResolvedValueOnce(json(503, {}));
    await expect(api.get("/me")).rejects.toBeInstanceOf(ApiError);
    expect(onExpired).not.toHaveBeenCalled();
  });

  it("turns network failures into a friendly ApiError and hides 5xx details", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("Network request failed"));
    await expect(api.get("/x")).rejects.toMatchObject({ status: 0, isNetwork: true });

    fetchMock.mockResolvedValueOnce(json(500, { error: "pq: syntax error at or near" }));
    const err = await api.get("/x").catch((e: ApiError) => e);
    expect((err as ApiError).message).not.toContain("pq:");
  });

  it("surfaces the server message on 4xx", async () => {
    fetchMock.mockResolvedValueOnce(json(409, { error: "email already exists" }));
    await expect(api.public("POST", "/auth/register", {})).rejects.toThrow("email already exists");
  });
});
