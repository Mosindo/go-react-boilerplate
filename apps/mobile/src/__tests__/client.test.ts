import { ApiError, configureSession, request, type Tokens } from "../api/client";

type FetchMock = jest.Mock<Promise<Response>, [string, RequestInit?]>;

function json(status: number, body?: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), { status });
}

describe("api client", () => {
  let tokens: Tokens | null;
  let lost: number;
  let fetchMock: FetchMock;

  beforeEach(() => {
    tokens = { accessToken: "old-access", refreshToken: "old-refresh" };
    lost = 0;
    configureSession({
      getTokens: () => tokens,
      setTokens: (t) => {
        tokens = t;
      },
      onAuthLost: () => {
        lost += 1;
      },
    });
    fetchMock = jest.fn();
    global.fetch = fetchMock as unknown as typeof fetch;
  });

  afterEach(() => configureSession(null));

  it("maps server errors to ApiError with code and message", async () => {
    fetchMock.mockResolvedValueOnce(
      json(409, { error: "email already exists", code: "email_exists" }),
    );
    await expect(
      request("/auth/register", { method: "POST", body: {}, auth: false }),
    ).rejects.toMatchObject({
      status: 409,
      code: "email_exists",
      message: "email already exists",
    });
  });

  it("reports network failures as status 0", async () => {
    fetchMock.mockRejectedValueOnce(new TypeError("Network request failed"));
    const err = await request("/me").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isNetwork).toBe(true);
  });

  it("refreshes once and retries when the access token expired", async () => {
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "invalid token" }))
      .mockResolvedValueOnce(json(200, { accessToken: "new-access", refreshToken: "new-refresh" }))
      .mockResolvedValueOnce(json(200, { id: "u1" }));
    await expect(request("/me")).resolves.toEqual({ id: "u1" });
    expect(tokens).toEqual({ accessToken: "new-access", refreshToken: "new-refresh" });
    const retryHeaders = fetchMock.mock.calls[2]?.[1]?.headers as Record<string, string>;
    expect(retryHeaders.Authorization).toBe("Bearer new-access");
  });

  it("shares a single refresh between concurrent requests", async () => {
    fetchMock.mockImplementation(async (url) => {
      if (url.endsWith("/auth/refresh"))
        return json(200, { accessToken: "new-access", refreshToken: "new-refresh" });
      const first = fetchMock.mock.calls.filter(([u]) => u === url).length === 1;
      return first ? json(401, { error: "expired" }) : json(200, { ok: true });
    });
    await Promise.all([request("/a"), request("/b"), request("/c")]);
    const refreshes = fetchMock.mock.calls.filter(([u]) => u.endsWith("/auth/refresh"));
    expect(refreshes).toHaveLength(1);
  });

  it("signs the user out when the refresh token is rejected", async () => {
    fetchMock
      .mockResolvedValueOnce(json(401, { error: "expired" }))
      .mockResolvedValueOnce(json(401, { error: "invalid refresh token" }));
    await expect(request("/me")).rejects.toMatchObject({ status: 401 });
    expect(lost).toBe(1);
  });

  it("does not try to refresh unauthenticated endpoints", async () => {
    fetchMock.mockResolvedValueOnce(
      json(401, { error: "invalid credentials", code: "invalid_credentials" }),
    );
    await expect(
      request("/auth/login", { method: "POST", body: {}, auth: false }),
    ).rejects.toMatchObject({ code: "invalid_credentials" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(lost).toBe(0);
  });

  it("handles 204 responses", async () => {
    fetchMock.mockResolvedValueOnce(json(204));
    await expect(request<void>("/blocks/1", { method: "DELETE" })).resolves.toBeUndefined();
  });
});
