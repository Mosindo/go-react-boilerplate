import { ApiError, NetworkError, getFreshAccessToken, getTokens, request, setTokens } from "../api/client";

function jwt(expSecondsFromNow: number): string {
  const payload = Buffer.from(
    JSON.stringify({ exp: Math.floor(Date.now() / 1000) + expSecondsFromNow })
  ).toString("base64url");
  return `h.${payload}.s`;
}

function respond(status: number, body?: unknown): Response {
  return new Response(body === undefined ? "" : JSON.stringify(body), { status });
}

const fetchMock = jest.fn();

beforeEach(() => {
  fetchMock.mockReset();
  (global as unknown as { fetch: typeof fetch }).fetch = fetchMock as unknown as typeof fetch;
  process.env.EXPO_PUBLIC_API_URL = "http://api.test";
  setTokens(null);
});

describe("api client", () => {
  it("sends the bearer token and parses JSON", async () => {
    setTokens({ accessToken: jwt(600), refreshToken: "r" });
    fetchMock.mockResolvedValueOnce(respond(200, { ok: true }));
    await expect(request<{ ok: boolean }>("/x")).resolves.toEqual({ ok: true });
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>).Authorization).toMatch(/^Bearer /);
  });

  it("surfaces API error codes", async () => {
    fetchMock.mockResolvedValueOnce(respond(409, { error: "email already exists", code: "email_taken" }));
    await expect(request("/auth/register", { method: "POST", body: {}, auth: false })).rejects.toMatchObject({
      status: 409,
      code: "email_taken"
    });
  });

  it("maps connection failures to NetworkError", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("Network request failed"));
    await expect(request("/x", { auth: false })).rejects.toBeInstanceOf(NetworkError);
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    setTokens({ accessToken: jwt(600), refreshToken: "old" });
    const rotated = jwt(900);
    fetchMock
      .mockResolvedValueOnce(respond(401, { error: "invalid token" }))
      .mockResolvedValueOnce(respond(200, { accessToken: rotated, refreshToken: "new", user: {} }))
      .mockResolvedValueOnce(respond(200, { done: true }));
    await expect(request("/x")).resolves.toEqual({ done: true });
    expect(getTokens()).toEqual({ accessToken: rotated, refreshToken: "new" });
    const retryInit = fetchMock.mock.calls[2]?.[1] as RequestInit;
    expect((retryInit.headers as Record<string, string>).Authorization).toBe(`Bearer ${rotated}`);
  });

  it("signs out when the refresh token is rejected", async () => {
    setTokens({ accessToken: jwt(600), refreshToken: "dead" });
    fetchMock
      .mockResolvedValueOnce(respond(401, { error: "x" }))
      .mockResolvedValueOnce(respond(401, { error: "invalid refresh token" }));
    await expect(request("/x")).rejects.toBeInstanceOf(ApiError);
    expect(getTokens()).toBeNull();
  });

  it("keeps the session on a network failure during refresh", async () => {
    setTokens({ accessToken: jwt(-10), refreshToken: "r" });
    fetchMock.mockRejectedValueOnce(new TypeError("offline"));
    await expect(getFreshAccessToken()).resolves.toBeNull();
    expect(getTokens()).not.toBeNull();
  });

  it("shares a single refresh between concurrent callers", async () => {
    setTokens({ accessToken: jwt(-10), refreshToken: "r" });
    const fresh = jwt(900);
    fetchMock.mockResolvedValue(respond(200, { accessToken: fresh, refreshToken: "r2", user: {} }));
    const [a, b] = await Promise.all([getFreshAccessToken(), getFreshAccessToken()]);
    expect(a).toBe(fresh);
    expect(b).toBe(fresh);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
