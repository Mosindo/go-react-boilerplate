import { api, ApiError } from "../api/client";
import { accessTokenTtl, session } from "../auth/session";

function jwt(expSecondsFromNow: number): string {
  const payload = Buffer.from(
    JSON.stringify({ exp: Math.floor(Date.now() / 1000) + expSecondsFromNow })
  ).toString("base64url");
  return `h.${payload}.s`;
}

function jsonResponse(status: number, body?: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), { status });
}

const fetchMock = jest.fn<Promise<Response>, [string, RequestInit?]>();

beforeEach(async () => {
  fetchMock.mockReset();
  global.fetch = fetchMock as unknown as typeof fetch;
  await session.clear();
});

describe("access token inspection", () => {
  it("reads the expiry from the JWT payload", () => {
    expect(accessTokenTtl(jwt(600))).toBeGreaterThan(590);
    expect(accessTokenTtl(jwt(-60))).toBeLessThan(0);
    expect(accessTokenTtl("garbage")).toBe(0);
  });
});

describe("api client", () => {
  it("sends the bearer token and parses JSON", async () => {
    await session.set({ accessToken: jwt(600), refreshToken: "r1" });
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { hello: "world" }));
    await expect(api<{ hello: string }>("/me")).resolves.toEqual({ hello: "world" });
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Record<string, string>;
    expect(headers.Authorization).toMatch(/^Bearer /);
  });

  it("maps API errors to ApiError with status and field", async () => {
    await session.set({ accessToken: jwt(600), refreshToken: "r1" });
    fetchMock.mockResolvedValueOnce(jsonResponse(400, { error: "bad date", field: "birthDate" }));
    const err = await api("/me/profile", { method: "PUT", body: {} }).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 400, message: "bad date", field: "birthDate" });
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    await session.set({ accessToken: jwt(600), refreshToken: "old-refresh" });
    const fresh = jwt(900);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(401, { error: "invalid token" }))
      .mockResolvedValueOnce(jsonResponse(200, { accessToken: fresh, refreshToken: "new-refresh" }))
      .mockResolvedValueOnce(jsonResponse(200, { ok: true }));
    await expect(api("/me")).resolves.toEqual({ ok: true });
    expect(fetchMock.mock.calls[1]?.[0]).toMatch(/\/auth\/refresh$/);
    expect(JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body))).toEqual({ refreshToken: "old-refresh" });
    expect(session.tokens?.refreshToken).toBe("new-refresh");
  });

  it("shares one refresh between concurrent requests", async () => {
    await session.set({ accessToken: jwt(5), refreshToken: "r" }); // about to expire: proactive refresh
    fetchMock.mockImplementation(async (url) => {
      if (String(url).endsWith("/auth/refresh"))
        return jsonResponse(200, { accessToken: jwt(900), refreshToken: "r2" });
      return jsonResponse(200, { ok: true });
    });
    await Promise.all([api("/a"), api("/b"), api("/c")]);
    const refreshCalls = fetchMock.mock.calls.filter(([url]) => String(url).endsWith("/auth/refresh"));
    expect(refreshCalls).toHaveLength(1);
  });

  it("signs the user out when the refresh token is rejected", async () => {
    await session.set({ accessToken: jwt(600), refreshToken: "revoked" });
    const listener = jest.fn();
    const unsubscribe = session.subscribe(listener);
    fetchMock
      .mockResolvedValueOnce(jsonResponse(401, { error: "invalid token" }))
      .mockResolvedValueOnce(jsonResponse(401, { error: "invalid refresh token" }));
    await expect(api("/me")).rejects.toMatchObject({ status: 401 });
    expect(session.tokens).toBeNull();
    expect(listener).toHaveBeenLastCalledWith(false);
    unsubscribe();
  });

  it("keeps the session on network failure and reports a friendly error", async () => {
    await session.set({ accessToken: jwt(600), refreshToken: "r" });
    fetchMock.mockRejectedValueOnce(new TypeError("Network request failed"));
    await expect(api("/me")).rejects.toMatchObject({ status: 0 });
    expect(session.tokens).not.toBeNull();
  });

  it("does not require a session for public endpoints", async () => {
    fetchMock.mockResolvedValueOnce(jsonResponse(200, { accessToken: "a", refreshToken: "b" }));
    await expect(api("/auth/login", { method: "POST", auth: false, body: {} })).resolves.toBeDefined();
    const headers = fetchMock.mock.calls[0]?.[1]?.headers as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
  });
});
