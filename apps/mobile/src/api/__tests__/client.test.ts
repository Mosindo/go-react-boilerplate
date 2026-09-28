import { ApiError, createApiClient, errorMessage, type AuthTokens, type TokenStore } from "../client";

function memoryTokens(initial: AuthTokens | null): TokenStore & { current: () => AuthTokens | null } {
  let tokens = initial;
  return {
    get: async () => tokens,
    save: async (next) => {
      tokens = next;
    },
    clear: async () => {
      tokens = null;
    },
    current: () => tokens
  };
}

function json(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(status === 204 ? null : JSON.stringify(body), { status, headers });
}

function jwt(expSeconds: number): string {
  const encode = (value: object) => btoa(JSON.stringify(value)).replace(/=+$/, "");
  return `${encode({ alg: "HS256" })}.${encode({ exp: expSeconds })}.sig`;
}

describe("api client", () => {
  it("sends the bearer token and JSON body, and parses JSON responses", async () => {
    const fetchImpl = jest.fn(async () => json(200, { ok: true }));
    const client = createApiClient({
      baseUrl: "http://api",
      fetchImpl,
      tokens: memoryTokens({ accessToken: "a1", refreshToken: "r1" })
    });
    const result = await client.request<{ ok: boolean }>("/me", {
      method: "PUT",
      body: { x: 1 },
      query: { limit: 5, skip: null }
    });
    expect(result).toEqual({ ok: true });
    const [url, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe("http://api/me?limit=5");
    expect(init.method).toBe("PUT");
    expect(init.body).toBe('{"x":1}');
    expect(init.headers).toMatchObject({ Authorization: "Bearer a1", "Content-Type": "application/json" });
  });

  it("does not attach a token to public requests and does not refresh on 401", async () => {
    const fetchImpl = jest.fn(async () => json(401, { error: "bad credentials", code: "unauthorized" }));
    const client = createApiClient({
      baseUrl: "http://api",
      fetchImpl,
      tokens: memoryTokens({ accessToken: "a1", refreshToken: "r1" })
    });
    await expect(client.request("/auth/login", { method: "POST", body: {}, auth: false })).rejects.toMatchObject({
      status: 401,
      message: "bad credentials"
    });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    const [, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect((init.headers as Record<string, string>).Authorization).toBeUndefined();
  });

  it("maps error payloads, retry-after and 204 responses", async () => {
    const responses = [
      json(429, { error: "slow down", code: "rate_limited" }, { "Retry-After": "12" }),
      json(422, { error: "too young", code: "underage" }),
      json(500, "boom"),
      json(204, null)
    ];
    const client = createApiClient({
      baseUrl: "http://api",
      fetchImpl: async () => responses.shift() as Response,
      tokens: memoryTokens(null)
    });
    await expect(client.request("/a", { auth: false })).rejects.toMatchObject({
      code: "rate_limited",
      retryAfterSeconds: 12
    });
    await expect(client.request("/b", { auth: false })).rejects.toMatchObject({ code: "underage", status: 422 });
    await expect(client.request("/c", { auth: false })).rejects.toMatchObject({ code: "internal", status: 500 });
    await expect(client.request("/d", { auth: false })).resolves.toBeUndefined();
  });

  it("reports network failures and timeouts as ApiErrors", async () => {
    const offline = createApiClient({
      baseUrl: "http://api",
      fetchImpl: async () => {
        throw new TypeError("Network request failed");
      },
      tokens: memoryTokens(null)
    });
    await expect(offline.request("/x", { auth: false })).rejects.toMatchObject({ code: "network", status: 0 });

    const slow = createApiClient({
      baseUrl: "http://api",
      timeoutMs: 5,
      fetchImpl: (_url, init) =>
        new Promise<Response>((_resolve, reject) => {
          (init as RequestInit).signal?.addEventListener("abort", () => reject(new Error("aborted")));
        }),
      tokens: memoryTokens(null)
    });
    await expect(slow.request("/x", { auth: false })).rejects.toMatchObject({ code: "timeout" });
  });

  it("fails clearly when no base URL is configured", async () => {
    const client = createApiClient({ baseUrl: "", fetchImpl: jest.fn(), tokens: memoryTokens(null) });
    await expect(client.request("/x", { auth: false })).rejects.toBeInstanceOf(ApiError);
  });

  describe("token refresh", () => {
    function setup(refreshResult: () => Response | Promise<Response>) {
      const tokens = memoryTokens({ accessToken: "old", refreshToken: "r-old" });
      const onSessionExpired = jest.fn();
      let refreshCalls = 0;
      const fetchImpl = jest.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (url.endsWith("/auth/refresh")) {
          refreshCalls += 1;
          return refreshResult();
        }
        const auth = (init?.headers as Record<string, string>).Authorization;
        return auth === "Bearer new" ? json(200, { path: url }) : json(401, { error: "expired", code: "unauthorized" });
      });
      const client = createApiClient({ baseUrl: "http://api", fetchImpl, tokens, onSessionExpired });
      return { client, tokens, onSessionExpired, fetchImpl, refreshCalls: () => refreshCalls };
    }

    const rotated = () =>
      json(200, { accessToken: "new", refreshToken: "r-new", user: { id: "u", email: "e", createdAt: "" } });

    it("refreshes once for many concurrent 401s (single flight) and retries each request", async () => {
      const ctx = setup(rotated);
      const results = await Promise.all([ctx.client.request("/a"), ctx.client.request("/b"), ctx.client.request("/c")]);
      expect(results).toHaveLength(3);
      expect(ctx.refreshCalls()).toBe(1);
      expect(ctx.tokens.current()).toEqual({ accessToken: "new", refreshToken: "r-new" });
      expect(ctx.onSessionExpired).not.toHaveBeenCalled();
    });

    it("reuses tokens already rotated by another caller instead of refreshing again", async () => {
      const ctx = setup(rotated);
      await ctx.client.request("/a");
      // A late request that still used the old token would 401, but tokens are already fresh.
      const stale = await ctx.client.request("/b");
      expect(stale).toBeDefined();
      expect(ctx.refreshCalls()).toBe(1);
    });

    it("signs the user out when the refresh token is rejected", async () => {
      const ctx = setup(() => json(401, { error: "revoked", code: "unauthorized" }));
      await expect(Promise.all([ctx.client.request("/a"), ctx.client.request("/b")])).rejects.toMatchObject({
        status: 401
      });
      expect(ctx.refreshCalls()).toBe(1);
      expect(ctx.tokens.current()).toBeNull();
      expect(ctx.onSessionExpired).toHaveBeenCalledTimes(1);
    });

    it("keeps the session when the refresh fails because of the network", async () => {
      const ctx = setup(() => {
        throw new TypeError("offline");
      });
      await expect(ctx.client.request("/a")).rejects.toMatchObject({ code: "network" });
      expect(ctx.tokens.current()).toEqual({ accessToken: "old", refreshToken: "r-old" });
      expect(ctx.onSessionExpired).not.toHaveBeenCalled();
    });

    it("does not loop when the retried request is still unauthorized", async () => {
      const tokens = memoryTokens({ accessToken: "old", refreshToken: "r" });
      const fetchImpl = jest.fn(async (input: RequestInfo | URL) =>
        String(input).endsWith("/auth/refresh")
          ? json(200, { accessToken: "new", refreshToken: "r2", user: {} })
          : json(401, { error: "nope", code: "unauthorized" })
      );
      const client = createApiClient({ baseUrl: "http://api", fetchImpl, tokens });
      await expect(client.request("/a")).rejects.toMatchObject({ status: 401 });
      expect(fetchImpl).toHaveBeenCalledTimes(3);
    });

    it("getValidAccessToken returns a fresh token as is and refreshes an expiring one", async () => {
      const nowMs = 1_000_000_000_000;
      const fresh = jwt(nowMs / 1000 + 600);
      const expiring = jwt(nowMs / 1000 + 5);
      const fetchImpl = jest.fn(async () =>
        json(200, { accessToken: "rotated", refreshToken: "r2", user: { id: "u", email: "e", createdAt: "" } })
      );

      const freshClient = createApiClient({
        baseUrl: "http://api",
        fetchImpl,
        now: () => nowMs,
        tokens: memoryTokens({ accessToken: fresh, refreshToken: "r" })
      });
      await expect(freshClient.getValidAccessToken()).resolves.toBe(fresh);
      expect(fetchImpl).not.toHaveBeenCalled();

      const expiringClient = createApiClient({
        baseUrl: "http://api",
        fetchImpl,
        now: () => nowMs,
        tokens: memoryTokens({ accessToken: expiring, refreshToken: "r" })
      });
      await expect(expiringClient.getValidAccessToken()).resolves.toBe("rotated");
      expect(fetchImpl).toHaveBeenCalledTimes(1);
    });

    it("getValidAccessToken is null when signed out", async () => {
      const client = createApiClient({ baseUrl: "http://api", fetchImpl: jest.fn(), tokens: memoryTokens(null) });
      await expect(client.getValidAccessToken()).resolves.toBeNull();
    });
  });
});

describe("error helpers", () => {
  it("errorMessage prefers the API message and softens rate limits", () => {
    expect(errorMessage(new ApiError(400, "invalid_request", "Bio too long"))).toBe("Bio too long");
    expect(errorMessage(new ApiError(429, "rate_limited", "x"))).toMatch(/wait/);
    expect(errorMessage(new Error("boom"), "fallback")).toBe("fallback");
  });
});
