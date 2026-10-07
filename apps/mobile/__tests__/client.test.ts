const mockStorage = new Map<string, string>();
jest.mock("expo-secure-store", () => ({
  setItemAsync: jest.fn(async (key: string, value: string) => {
    mockStorage.set(key, value);
  }),
  getItemAsync: jest.fn(async (key: string) => mockStorage.get(key) ?? null),
  deleteItemAsync: jest.fn(async (key: string) => {
    mockStorage.delete(key);
  })
}));

import {
  API_BASE_URL,
  apiRequest,
  ApiError,
  resetClientStateForTests,
  setSessionExpiredHandler
} from "../src/api/client";
import { resetFeedback } from "../src/shared/feedback";
import { clearTokens, getTokens, saveTokens } from "../src/store/tokenStore";

type Call = { url: string; init: RequestInit };
let calls: Call[];
let responder: (call: Call) => Response | Promise<Response>;

function json(status: number, body?: unknown, headers: Record<string, string> = {}): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json", ...headers }
  });
}

function authHeader(call: Call): string | undefined {
  return (call.init.headers as Record<string, string>).Authorization;
}

const session = { accessToken: "new-access", refreshToken: "new-refresh", user: { id: "u1" } };

beforeEach(async () => {
  calls = [];
  mockStorage.clear();
  resetFeedback();
  resetClientStateForTests();
  await clearTokens();
  globalThis.fetch = jest.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    const call = { url: String(url), init: init ?? {} };
    calls.push(call);
    return responder(call);
  }) as unknown as typeof fetch;
});

describe("apiRequest", () => {
  it("sends the bearer token and JSON body", async () => {
    await saveTokens({ accessToken: "tok", refreshToken: "ref" });
    responder = () => json(200, { ok: true });
    const result = await apiRequest<{ ok: boolean }>("/me", { method: "POST", body: { a: 1 } });
    expect(result).toEqual({ ok: true });
    expect(authHeader(calls[0])).toBe("Bearer tok");
    expect(calls[0].init.body).toBe('{"a":1}');
    expect((calls[0].init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("does not attach a token to public routes and does not refresh on their 401", async () => {
    await saveTokens({ accessToken: "tok", refreshToken: "ref" });
    responder = () => json(401, { error: "bad credentials", code: "unauthorized" });
    await expect(apiRequest("/auth/login", { method: "POST", body: {}, auth: false })).rejects.toMatchObject({
      status: 401,
      code: "unauthorized",
      message: "bad credentials"
    });
    expect(calls).toHaveLength(1);
    expect(authHeader(calls[0])).toBeUndefined();
  });

  it("does not force a JSON content type on FormData", async () => {
    responder = () => json(201, { id: "p" });
    const form = new FormData();
    form.append("field", "value");
    await apiRequest("/photos", { method: "POST", body: form });
    expect((calls[0].init.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
    expect(calls[0].init.body).toBe(form);
  });

  it("returns undefined for 204", async () => {
    responder = () => new Response(null, { status: 204 });
    await expect(apiRequest("/x", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("refreshes once on 401 and retries with the new token", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "ref" });
    responder = (call) => {
      if (call.url.endsWith("/auth/refresh")) return json(200, session);
      return authHeader(call) === "Bearer new-access"
        ? json(200, { data: 1 })
        : json(401, { error: "expired" });
    };
    await expect(apiRequest("/profile/me")).resolves.toEqual({ data: 1 });
    expect(calls.map((c) => c.url.slice(API_BASE_URL.length))).toEqual([
      "/profile/me",
      "/auth/refresh",
      "/profile/me"
    ]);
    expect(JSON.parse(String(calls[1].init.body))).toEqual({ refreshToken: "ref" });
    expect(await getTokens()).toEqual({ accessToken: "new-access", refreshToken: "new-refresh" });
  });

  it("shares one refresh between concurrent 401s", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "ref" });
    responder = (call) => {
      if (call.url.endsWith("/auth/refresh")) return json(200, session);
      return authHeader(call) === "Bearer new-access" ? json(200, {}) : json(401, {});
    };
    await Promise.all([apiRequest("/a"), apiRequest("/b"), apiRequest("/c")]);
    expect(calls.filter((c) => c.url.endsWith("/auth/refresh"))).toHaveLength(1);
  });

  it("retries only once when the retried request is still 401", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "ref" });
    responder = (call) =>
      call.url.endsWith("/auth/refresh") ? json(200, session) : json(401, { error: "no" });
    await expect(apiRequest("/me", { method: "DELETE", body: { password: "x" } })).rejects.toMatchObject({
      status: 401
    });
    expect(calls).toHaveLength(3);
    // Session stays: the refresh itself succeeded.
    expect(await getTokens()).not.toBeNull();
  });

  it("clears the session and notifies when the refresh token is rejected", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "ref" });
    const expired = jest.fn();
    setSessionExpiredHandler(expired);
    responder = (call) =>
      call.url.endsWith("/auth/refresh") ? json(401, { error: "revoked" }) : json(401, { error: "expired" });
    await expect(apiRequest("/me")).rejects.toBeInstanceOf(ApiError);
    expect(expired).toHaveBeenCalledTimes(1);
    expect(await getTokens()).toBeNull();
  });

  it("keeps the session when the refresh fails transiently", async () => {
    await saveTokens({ accessToken: "old", refreshToken: "ref" });
    const expired = jest.fn();
    setSessionExpiredHandler(expired);
    responder = (call) => (call.url.endsWith("/auth/refresh") ? json(503, { error: "down" }) : json(401, {}));
    await expect(apiRequest("/me")).rejects.toMatchObject({ status: 503 });
    expect(expired).not.toHaveBeenCalled();
    expect(await getTokens()).toEqual({ accessToken: "old", refreshToken: "ref" });
  });

  it("maps error payloads, defaults and Retry-After", async () => {
    responder = () => json(429, { error: "slow down", code: "rate_limited" }, { "Retry-After": "7" });
    await expect(apiRequest("/x", { auth: false })).rejects.toMatchObject({
      status: 429,
      code: "rate_limited",
      retryAfterSeconds: 7
    });
    responder = () => new Response("oops", { status: 404 });
    await expect(apiRequest("/x", { auth: false })).rejects.toMatchObject({ status: 404, code: "not_found" });
  });

  it("turns network failures into a network_error ApiError", async () => {
    globalThis.fetch = jest.fn(async () => {
      throw new TypeError("Network request failed");
    }) as unknown as typeof fetch;
    await expect(apiRequest("/x", { auth: false })).rejects.toMatchObject({
      code: "network_error",
      status: 0
    });
  });
});
