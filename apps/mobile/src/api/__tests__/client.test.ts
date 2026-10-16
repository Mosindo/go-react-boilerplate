import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("react-native", () => ({ Platform: { OS: "ios" } }));
vi.mock("../../shared/feedback", () => ({ showGlobalError: vi.fn() }));

const tokenState = vi.hoisted(() => ({ current: null as { accessToken: string; refreshToken: string } | null }));
vi.mock("../../store/tokenStore", () => ({
  getTokens: async () => tokenState.current,
  saveTokens: async (tokens: { accessToken: string; refreshToken: string }) => {
    tokenState.current = tokens;
  },
  clearTokens: async () => {
    tokenState.current = null;
  }
}));

// Imports are hoisted: the environment the client reads at load time must be set in a hoisted block.
vi.hoisted(() => {
  (globalThis as { __DEV__?: boolean }).__DEV__ = true;
  process.env.EXPO_PUBLIC_API_URL = "http://api.test";
});

import { ApiError, apiRequest, onSessionExpired } from "../client";
import { friendlyError } from "../errors";

type Call = { url: string; auth: string | null };
let calls: Call[] = [];

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
}

function stubFetch(handler: (url: string, auth: string | null) => Response) {
  calls = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init: RequestInit) => {
      const auth = new Headers(init.headers).get("Authorization");
      calls.push({ url, auth });
      return handler(url, auth);
    })
  );
}

beforeEach(() => {
  tokenState.current = { accessToken: "old-access", refreshToken: "old-refresh" };
});

describe("apiRequest", () => {
  it("sends the bearer token and parses JSON", async () => {
    stubFetch(() => json(200, { ok: true }));
    await expect(apiRequest("/me")).resolves.toEqual({ ok: true });
    expect(calls[0]).toEqual({ url: "http://api.test/me", auth: "Bearer old-access" });
  });

  it("refreshes once for concurrent 401s, then retries with the new token", async () => {
    stubFetch((url, auth) => {
      if (url.endsWith("/auth/refresh")) {
        return json(200, { accessToken: "new-access", refreshToken: "new-refresh" });
      }
      return auth === "Bearer new-access" ? json(200, { done: true }) : json(401, { error: "invalid token" });
    });

    const results = await Promise.all([apiRequest("/a"), apiRequest("/b"), apiRequest("/c")]);
    expect(results).toEqual([{ done: true }, { done: true }, { done: true }]);
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
    expect(tokenState.current).toEqual({ accessToken: "new-access", refreshToken: "new-refresh" });
  });

  it("ends the session when the refresh token is rejected", async () => {
    stubFetch((url) =>
      url.endsWith("/auth/refresh") ? json(401, { error: "invalid refresh token" }) : json(401, { error: "invalid token" })
    );
    const expired = vi.fn();
    const off = onSessionExpired(expired);

    await expect(apiRequest("/me")).rejects.toMatchObject({ status: 401 });
    expect(expired).toHaveBeenCalledTimes(1);
    expect(tokenState.current).toBeNull();
    off();
  });

  it("does not refresh for auth endpoints (a wrong password is not an expired session)", async () => {
    stubFetch(() => json(401, { error: "invalid credentials" }));
    await expect(apiRequest("/auth/login", { method: "POST", body: "{}", skipAuthRefresh: true })).rejects.toBeInstanceOf(ApiError);
    expect(calls).toHaveLength(1);
    expect(tokenState.current).not.toBeNull();
  });
});

describe("friendlyError", () => {
  it("translates known server messages and falls back by status", () => {
    expect(friendlyError(new ApiError(401, { error: "invalid credentials" }))).toBe("E-mail ou mot de passe incorrect.");
    expect(friendlyError(new ApiError(409, { error: "email already exists" }))).toBe("Cette adresse e-mail est déjà utilisée.");
    expect(friendlyError(new ApiError(429, { error: "too many requests, please slow down" }))).toMatch(/Trop de/);
    expect(friendlyError(new ApiError(500, null))).toMatch(/indisponible/);
    expect(friendlyError(new Error("custom"))).toBe("custom");
  });
});
