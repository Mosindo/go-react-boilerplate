import { ApiError, request } from "./client";
import { clearTokens, getTokensSync, saveTokens } from "./session";

jest.mock("../config", () => ({ API_BASE_URL: "http://api.test" }));

type Reply = { status: number; body?: unknown };

function mockFetch(replies: Reply[]) {
  const calls: { url: string; auth: string | null }[] = [];
  const fn = jest.fn(async (url: string, init: RequestInit) => {
    const headers = init.headers as Record<string, string>;
    calls.push({ url, auth: headers.Authorization ?? null });
    const reply = replies.shift();
    if (!reply) throw new Error("unexpected request " + url);
    return {
      ok: reply.status >= 200 && reply.status < 300,
      status: reply.status,
      json: async () => reply.body,
      text: async () => (reply.body === undefined ? "" : JSON.stringify(reply.body))
    } as Response;
  });
  global.fetch = fn as unknown as typeof fetch;
  return calls;
}

describe("api client", () => {
  beforeEach(async () => {
    await saveTokens({ accessToken: "old-access", refreshToken: "old-refresh" });
  });
  afterEach(async () => {
    await clearTokens();
  });

  it("sends the bearer token and parses JSON", async () => {
    const calls = mockFetch([{ status: 200, body: { id: "u1" } }]);
    await expect(request("/me")).resolves.toEqual({ id: "u1" });
    expect(calls[0]).toEqual({ url: "http://api.test/me", auth: "Bearer old-access" });
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    const calls = mockFetch([
      { status: 401, body: { error: "invalid token" } },
      { status: 200, body: { accessToken: "new-access", refreshToken: "new-refresh", user: {} } },
      { status: 200, body: { ok: true } }
    ]);
    await expect(request("/profile")).resolves.toEqual({ ok: true });
    expect(calls.map((c) => c.url)).toEqual([
      "http://api.test/profile",
      "http://api.test/auth/refresh",
      "http://api.test/profile"
    ]);
    expect(calls[2]?.auth).toBe("Bearer new-access");
    expect(getTokensSync()).toEqual({ accessToken: "new-access", refreshToken: "new-refresh" });
  });

  it("signs the user out when the refresh token is rejected", async () => {
    mockFetch([
      { status: 401, body: { error: "invalid token" } },
      { status: 401, body: { error: "invalid refresh token" } }
    ]);
    await expect(request("/profile")).rejects.toMatchObject({ status: 401 });
    expect(getTokensSync()).toBeNull();
  });

  it("shares one refresh between concurrent requests", async () => {
    const calls = mockFetch([
      { status: 401 },
      { status: 401 },
      { status: 200, body: { accessToken: "a2", refreshToken: "r2", user: {} } },
      { status: 200, body: { n: 1 } },
      { status: 200, body: { n: 2 } }
    ]);
    const [a, b] = await Promise.all([request("/a"), request("/b")]);
    expect([a, b]).toEqual([{ n: 1 }, { n: 2 }]);
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
  });

  it("surfaces server error messages and handles 204", async () => {
    mockFetch([{ status: 409, body: { error: "an account with this email already exists" } }]);
    const error = await request("/auth/register", { method: "POST", body: {}, auth: false }).catch(
      (e: unknown) => e
    );
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).message).toBe("Un compte existe déjà avec cette adresse e-mail.");

    mockFetch([{ status: 204 }]);
    await expect(request("/blocks/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("does not attach credentials to unauthenticated calls", async () => {
    const calls = mockFetch([{ status: 200, body: {} }]);
    await request("/auth/login", { method: "POST", body: {}, auth: false });
    expect(calls[0]?.auth).toBeNull();
  });
});
