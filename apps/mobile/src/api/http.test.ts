import { ApiError, createHttpClient, readTokenExpiry, type TokenPair, type TokenStorage } from "./http";

function memoryStorage(initial: TokenPair | null): TokenStorage & { current: TokenPair | null } {
  const storage = {
    current: initial,
    get: () => storage.current,
    set: (tokens: TokenPair) => {
      storage.current = tokens;
      return Promise.resolve();
    },
    clear: () => {
      storage.current = null;
      return Promise.resolve();
    }
  };
  return storage;
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" }
  });
}

function jwt(exp: number): string {
  const encode = (value: object) => btoa(JSON.stringify(value)).replace(/=+$/, "");
  return `${encode({ alg: "HS256" })}.${encode({ exp })}.sig`;
}

type FetchMock = jest.Mock<Promise<Response>, [RequestInfo | URL, RequestInit | undefined]>;

function setup(options: {
  tokens?: TokenPair | null;
  handler: (url: string, init: RequestInit | undefined) => Response | Promise<Response>;
}) {
  const storage = memoryStorage(
    options.tokens === undefined ? { accessToken: "old", refreshToken: "r1" } : options.tokens
  );
  const fetchImpl: FetchMock = jest.fn((input, init) => Promise.resolve(options.handler(String(input), init)));
  const onSessionExpired = jest.fn();
  const client = createHttpClient({
    baseUrl: "http://api.test",
    storage,
    fetchImpl: fetchImpl as unknown as typeof fetch,
    onSessionExpired
  });
  return { client, storage, fetchImpl, onSessionExpired };
}

const authHeader = (init: RequestInit | undefined) => (init?.headers as Record<string, string>).Authorization;

describe("error mapping", () => {
  it("maps {error, code} payloads to ApiError", async () => {
    const { client } = setup({ handler: () => jsonResponse(409, { error: "taken", code: "email_taken" }) });
    const error = await client.get("/x").catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 409, code: "email_taken", message: "taken" });
  });

  it("derives a code from the status when the body has none", async () => {
    const { client } = setup({ handler: () => new Response("boom", { status: 500 }) });
    await expect(client.get("/x")).rejects.toMatchObject({ status: 500, code: "internal" });
  });

  it("reports network failures as network_error", async () => {
    const { client, fetchImpl } = setup({ handler: () => jsonResponse(200, {}) });
    fetchImpl.mockRejectedValueOnce(new TypeError("Network request failed"));
    await expect(client.get("/x")).rejects.toMatchObject({ status: 0, code: "network_error" });
  });

  it("reports timeouts", async () => {
    jest.useFakeTimers();
    try {
      const { client, fetchImpl } = setup({ handler: () => jsonResponse(200, {}) });
      fetchImpl.mockImplementationOnce(
        (_input, init) =>
          new Promise((_resolve, reject) => {
            init?.signal?.addEventListener("abort", () => reject(new Error("aborted")));
          })
      );
      const pending = client.get("/slow").catch((e: unknown) => e);
      await jest.advanceTimersByTimeAsync(10_000);
      expect(await pending).toMatchObject({ code: "timeout" });
    } finally {
      jest.useRealTimers();
    }
  });

  it("returns undefined for empty 204 bodies and serialises JSON/query", async () => {
    const { client, fetchImpl } = setup({ handler: () => new Response(null, { status: 204 }) });
    await expect(client.post("/things", { a: 1 })).resolves.toBeNull();
    await client.get("/list", { limit: 5, cursor: undefined, q: "a b" });
    expect(fetchImpl.mock.calls[0]?.[1]?.body).toBe('{"a":1}');
    expect(fetchImpl.mock.calls[1]?.[0]).toBe("http://api.test/list?limit=5&q=a%20b");
  });
});

describe("token refresh", () => {
  it("refreshes once for concurrent 401s and retries each request with the new token", async () => {
    let refreshCalls = 0;
    const { client, storage, fetchImpl } = setup({
      handler: (url, init) => {
        if (url.endsWith("/auth/refresh")) {
          refreshCalls += 1;
          return jsonResponse(200, { accessToken: "new", refreshToken: "r2", user: {} });
        }
        return authHeader(init) === "Bearer new"
          ? jsonResponse(200, { ok: true })
          : jsonResponse(401, { error: "expired", code: "unauthorized" });
      }
    });

    const results = await Promise.all([client.get("/a"), client.get("/b"), client.get("/c")]);

    expect(results).toEqual([{ ok: true }, { ok: true }, { ok: true }]);
    expect(refreshCalls).toBe(1);
    expect(storage.current).toEqual({ accessToken: "new", refreshToken: "r2" });
    expect(fetchImpl).toHaveBeenCalledTimes(7);
  });

  it("does not refresh again when another request already rotated the token", async () => {
    let refreshCalls = 0;
    const { client } = setup({
      handler: (url, init) => {
        if (url.endsWith("/auth/refresh")) {
          refreshCalls += 1;
          return jsonResponse(200, { accessToken: "new", refreshToken: "r2", user: {} });
        }
        return authHeader(init) === "Bearer new"
          ? jsonResponse(200, {})
          : jsonResponse(401, { error: "x", code: "unauthorized" });
      }
    });
    await client.get("/first");
    await client.get("/second");
    expect(refreshCalls).toBe(1);
  });

  it("signs out when the retry is still unauthorized", async () => {
    const { client, storage, onSessionExpired } = setup({
      handler: (url) =>
        url.endsWith("/auth/refresh")
          ? jsonResponse(200, { accessToken: "new", refreshToken: "r2", user: {} })
          : jsonResponse(401, { error: "nope", code: "unauthorized" })
    });
    await expect(client.get("/a")).rejects.toMatchObject({ status: 401 });
    expect(storage.current).toBeNull();
    expect(onSessionExpired).toHaveBeenCalledTimes(1);
  });

  it("signs out when the refresh token is rejected", async () => {
    const { client, storage, onSessionExpired } = setup({
      handler: (url) =>
        url.endsWith("/auth/refresh")
          ? jsonResponse(401, { error: "bad", code: "unauthorized" })
          : jsonResponse(401, { error: "x", code: "unauthorized" })
    });
    await expect(client.get("/a")).rejects.toBeInstanceOf(ApiError);
    expect(storage.current).toBeNull();
    expect(onSessionExpired).toHaveBeenCalledTimes(1);
  });

  it("keeps the session when the refresh fails because the network is down", async () => {
    const { client, storage, onSessionExpired, fetchImpl } = setup({
      handler: () => jsonResponse(401, { error: "x", code: "unauthorized" })
    });
    fetchImpl.mockImplementationOnce(() => Promise.resolve(jsonResponse(401, { error: "x", code: "unauthorized" })));
    fetchImpl.mockRejectedValueOnce(new TypeError("offline"));
    await expect(client.get("/a")).rejects.toMatchObject({ code: "network_error" });
    expect(storage.current).not.toBeNull();
    expect(onSessionExpired).not.toHaveBeenCalled();
  });

  it("treats invalid_credentials as a normal error, not an expired session", async () => {
    const { client, storage, onSessionExpired, fetchImpl } = setup({
      handler: () => jsonResponse(401, { error: "wrong password", code: "invalid_credentials" })
    });
    await expect(client.delete("/me", { password: "x" })).rejects.toMatchObject({ code: "invalid_credentials" });
    expect(fetchImpl).toHaveBeenCalledTimes(1);
    expect(storage.current).not.toBeNull();
    expect(onSessionExpired).not.toHaveBeenCalled();
  });

  it("never attaches credentials to auth:false requests", async () => {
    const { client, fetchImpl } = setup({ handler: () => jsonResponse(200, {}) });
    await client.post("/auth/login", { email: "a@b.co", password: "x" }, { auth: false });
    expect((fetchImpl.mock.calls[0]?.[1]?.headers as Record<string, string>).Authorization).toBeUndefined();
  });
});

describe("getFreshAccessToken", () => {
  it("returns a valid token untouched and refreshes one about to expire", async () => {
    const now = 1_700_000_000_000;
    const valid = jwt(now / 1000 + 600);
    const expiring = jwt(now / 1000 + 5);
    const storage = memoryStorage({ accessToken: valid, refreshToken: "r1" });
    const withClock = createHttpClient({
      baseUrl: "http://api.test",
      storage,
      fetchImpl: (() =>
        Promise.resolve(
          jsonResponse(200, { accessToken: "renewed", refreshToken: "r2", user: {} })
        )) as unknown as typeof fetch,
      now: () => now
    });
    expect(await withClock.getFreshAccessToken()).toBe(valid);
    storage.current = { accessToken: expiring, refreshToken: "r1" };
    expect(await withClock.getFreshAccessToken()).toBe("renewed");
  });

  it("decodes JWT expiry and tolerates garbage", () => {
    expect(readTokenExpiry(jwt(123))).toBe(123);
    expect(readTokenExpiry("garbage")).toBeNull();
  });
});
