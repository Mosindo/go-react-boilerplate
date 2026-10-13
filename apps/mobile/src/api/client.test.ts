import { API_BASE_URL, ApiError, apiRequest, errorMessage, isApiError } from "./client";
import { clearTokens, peekTokens, saveTokens } from "../store/tokenStore";

jest.mock("../shared/feedback/store", () => ({ showGlobalError: jest.fn() }));

type FetchCall = { url: string; headers: Record<string, string> };

function mockFetch(responses: { status: number; body?: unknown }[]) {
  const calls: FetchCall[] = [];
  global.fetch = jest.fn(async (url: string, init?: { headers?: Record<string, string> }) => {
    calls.push({ url, headers: init?.headers ?? {} });
    const next = responses.shift();
    if (!next) {
      throw new TypeError("network down");
    }
    return {
      ok: next.status >= 200 && next.status < 300,
      status: next.status,
      json: async () => next.body,
      text: async () => (next.body === undefined ? "" : JSON.stringify(next.body))
    };
  }) as unknown as typeof fetch;
  return calls;
}

describe("apiRequest", () => {
  beforeEach(async () => {
    await clearTokens();
  });

  it("sends the bearer token and parses JSON", async () => {
    await saveTokens({ accessToken: "access-1", refreshToken: "refresh-1" });
    const calls = mockFetch([{ status: 200, body: { ok: true } }]);
    await expect(apiRequest("/me")).resolves.toEqual({ ok: true });
    expect(calls[0].headers.Authorization).toBe("Bearer access-1");
  });

  it("turns API errors into ApiError with the server message", async () => {
    mockFetch([{ status: 409, body: { error: "email already exists" } }]);
    const error = await apiRequest("/auth/register", { method: "POST", json: {}, anonymous: true }).catch((e) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(isApiError(error, 409)).toBe(true);
    expect(errorMessage(error)).toBe("email already exists");
  });

  it("refreshes the session once on 401 and retries the request", async () => {
    await saveTokens({ accessToken: "expired", refreshToken: "refresh-1" });
    const calls = mockFetch([
      { status: 401, body: { error: "invalid token" } },
      { status: 200, body: { accessToken: "access-2", refreshToken: "refresh-2" } },
      { status: 200, body: { id: "u1" } }
    ]);
    await expect(apiRequest("/me")).resolves.toEqual({ id: "u1" });
    expect(calls.map((c) => c.url)).toEqual([`${API_BASE_URL}/me`, `${API_BASE_URL}/auth/refresh`, `${API_BASE_URL}/me`]);
    expect(calls[2].headers.Authorization).toBe("Bearer access-2");
    expect(peekTokens()?.refreshToken).toBe("refresh-2");
  });

  it("signs the user out when the refresh token is rejected", async () => {
    await saveTokens({ accessToken: "expired", refreshToken: "revoked" });
    mockFetch([
      { status: 401, body: { error: "invalid token" } },
      { status: 401, body: { error: "invalid refresh token" } }
    ]);
    await expect(apiRequest("/me")).rejects.toMatchObject({ status: 401 });
    expect(peekTokens()).toBeNull();
  });

  it("reports network failures as a status-0 ApiError", async () => {
    mockFetch([]);
    await expect(apiRequest("/health", { anonymous: true })).rejects.toMatchObject({ status: 0 });
  });
});
