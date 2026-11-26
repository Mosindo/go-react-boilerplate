import { api, ApiError } from "./client";
import { clearTokens, currentTokens, saveTokens } from "../store/tokenStore";

jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn(async () => null),
  setItemAsync: jest.fn(async () => undefined),
  deleteItemAsync: jest.fn(async () => undefined)
}));

type FetchCall = { url: string; init: RequestInit };

function mockFetch(responder: (call: FetchCall) => { status: number; body?: unknown }) {
  const calls: FetchCall[] = [];
  global.fetch = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    const call = { url: String(url), init: init ?? {} };
    calls.push(call);
    const { status, body } = responder(call);
    return new Response(body === undefined ? "" : JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return calls;
}

function authHeader(call: FetchCall): string | undefined {
  return (call.init.headers as Record<string, string> | undefined)?.Authorization;
}

describe("api client", () => {
  beforeEach(async () => {
    await clearTokens();
  });

  it("sends the bearer token and parses JSON", async () => {
    await saveTokens({ accessToken: "a1", refreshToken: "r1" });
    const calls = mockFetch(() => ({ status: 200, body: { ok: true } }));
    await expect(api<{ ok: boolean }>("/me")).resolves.toEqual({ ok: true });
    expect(authHeader(calls[0])).toBe("Bearer a1");
  });

  it("refreshes once on 401 then retries with the new token", async () => {
    await saveTokens({ accessToken: "expired", refreshToken: "r1" });
    const calls = mockFetch((call) => {
      if (call.url.endsWith("/auth/refresh")) return { status: 200, body: { accessToken: "fresh", refreshToken: "r2" } };
      return authHeader(call) === "Bearer fresh"
        ? { status: 200, body: { ok: 1 } }
        : { status: 401, body: { error: "expired", code: "invalid_token" } };
    });
    const results = await Promise.all([api("/a"), api("/b")]);
    expect(results).toEqual([{ ok: 1 }, { ok: 1 }]);
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
    expect(currentTokens()).toEqual({ accessToken: "fresh", refreshToken: "r2" });
  });

  it("clears the session when the refresh token is rejected", async () => {
    await saveTokens({ accessToken: "expired", refreshToken: "revoked" });
    mockFetch(() => ({ status: 401, body: { error: "Session expirée.", code: "invalid_refresh_token" } }));
    await expect(api("/me")).rejects.toBeInstanceOf(ApiError);
    expect(currentTokens()).toBeNull();
  });

  it("maps server errors and network failures to ApiError", async () => {
    mockFetch(() => ({ status: 409, body: { error: "Déjà fait.", code: "already_swiped" } }));
    await expect(api("/swipes", { method: "POST", body: {} })).rejects.toMatchObject({
      status: 409,
      code: "already_swiped",
      message: "Déjà fait."
    });

    global.fetch = jest.fn(async () => {
      throw new TypeError("Network request failed");
    }) as unknown as typeof fetch;
    await expect(api("/me")).rejects.toMatchObject({ code: "network_error" });
  });
});
