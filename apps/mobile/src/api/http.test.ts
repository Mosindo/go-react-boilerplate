import { ApiError, createApiClient } from './http';
import { createTokenStore, type TokenStore } from '../store/tokenStore';

type Reply = { status: number; body?: unknown; headers?: Record<string, string> };

function fakeResponse({ status, body, headers = {} }: Reply): Response {
  const text = body === undefined ? '' : JSON.stringify(body);
  return {
    status,
    ok: status >= 200 && status < 300,
    headers: { get: (k: string) => headers[k] ?? null },
    text: async () => text,
    json: async () => JSON.parse(text),
    blob: async () => ({ size: text.length }) as unknown as Blob,
  } as unknown as Response;
}

function memoryStore(initial: { accessToken: string; refreshToken: string } | null): TokenStore {
  let value = initial ? JSON.stringify(initial) : null;
  return createTokenStore({
    read: async () => value,
    write: async (v) => {
      value = v;
    },
    remove: async () => {
      value = null;
    },
  });
}

type Call = { url: string; method: string; auth: string | null; body: unknown };

function setup(
  handler: (call: Call) => Reply | Promise<Reply>,
  initial: { accessToken: string; refreshToken: string } | null = {
    accessToken: 'A1',
    refreshToken: 'R1',
  },
) {
  const calls: Call[] = [];
  const tokens = memoryStore(initial);
  const fetchImpl = jest.fn(async (url: string | URL | Request, init?: RequestInit) => {
    const headers = (init?.headers ?? {}) as Record<string, string>;
    const call: Call = {
      url: String(url),
      method: init?.method ?? 'GET',
      auth: headers.Authorization ?? null,
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : init?.body,
    };
    calls.push(call);
    return fakeResponse(await handler(call));
  }) as unknown as typeof fetch;
  const client = createApiClient({ baseUrl: 'http://api.test', tokens, fetchImpl });
  const onLost = jest.fn();
  client.setAuthLostHandler(onLost);
  return { client, calls, tokens, onLost };
}

describe('api client', () => {
  it('sends the bearer token, JSON body and query string', async () => {
    const { client, calls } = setup(() => ({ status: 200, body: { ok: true } }));
    const res = await client.post<{ ok: boolean }>('/swipes', { userId: 'u', action: 'like' });
    expect(res).toEqual({ ok: true });
    await client.get('/discover', { query: { limit: 10, before: undefined, q: 'a b' } });
    expect(calls[0]).toMatchObject({
      url: 'http://api.test/swipes',
      method: 'POST',
      auth: 'Bearer A1',
    });
    expect(calls[0]?.body).toEqual({ userId: 'u', action: 'like' });
    expect(calls[1]?.url).toBe('http://api.test/discover?limit=10&q=a%20b');
  });

  it('returns undefined for 204 responses', async () => {
    const { client } = setup(() => ({ status: 204 }));
    await expect(client.post('/conversations/1/read')).resolves.toBeUndefined();
  });

  it('parses API errors into ApiError with code, status and Retry-After', async () => {
    const { client } = setup(() => ({
      status: 429,
      body: { error: 'slow down', code: 'rate_limited' },
      headers: { 'Retry-After': '12' },
    }));
    const err = await client.get('/me').catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({
      status: 429,
      code: 'rate_limited',
      retryAfter: 12,
      message: 'slow down',
    });
  });

  it('does not send auth or try to refresh when auth is disabled', async () => {
    const { client, calls } = setup(() => ({
      status: 401,
      body: { error: 'bad', code: 'unauthorized' },
    }));
    await expect(
      client.post('/auth/login', { email: 'a', password: 'b' }, { auth: false }),
    ).rejects.toMatchObject({
      status: 401,
    });
    expect(calls).toHaveLength(1);
    expect(calls[0]?.auth).toBeNull();
  });

  it('refreshes once on 401, stores the rotated tokens and retries', async () => {
    let apiCalls = 0;
    const { client, calls, tokens, onLost } = setup((call) => {
      if (call.url.endsWith('/auth/refresh')) {
        return { status: 200, body: { accessToken: 'A2', refreshToken: 'R2', user: {} } };
      }
      apiCalls += 1;
      return call.auth === 'Bearer A2' ? { status: 200, body: { id: 'me' } } : { status: 401 };
    });
    await expect(client.get('/me')).resolves.toEqual({ id: 'me' });
    expect(calls.map((c) => c.url.replace('http://api.test', ''))).toEqual([
      '/me',
      '/auth/refresh',
      '/me',
    ]);
    expect(calls[1]?.body).toEqual({ refreshToken: 'R1' });
    expect(apiCalls).toBe(2);
    expect(await tokens.get()).toEqual({ accessToken: 'A2', refreshToken: 'R2' });
    expect(onLost).not.toHaveBeenCalled();
  });

  it('single-flights concurrent refreshes (rotating refresh tokens)', async () => {
    let refreshes = 0;
    const { client } = setup(async (call) => {
      if (call.url.endsWith('/auth/refresh')) {
        refreshes += 1;
        await new Promise((r) => setTimeout(r, 10));
        return { status: 200, body: { accessToken: 'A2', refreshToken: 'R2', user: {} } };
      }
      return call.auth === 'Bearer A2' ? { status: 200, body: {} } : { status: 401 };
    });
    await Promise.all([client.get('/a'), client.get('/b'), client.get('/c')]);
    expect(refreshes).toBe(1);
  });

  it('logs out (clears tokens, notifies) when the refresh is rejected', async () => {
    const { client, tokens, onLost, calls } = setup((call) =>
      call.url.endsWith('/auth/refresh')
        ? { status: 401, body: { code: 'unauthorized' } }
        : { status: 401 },
    );
    await expect(client.get('/me')).rejects.toMatchObject({ status: 401, code: 'unauthorized' });
    expect(onLost).toHaveBeenCalledTimes(1);
    expect(await tokens.get()).toBeNull();
    expect(calls.filter((c) => c.url.endsWith('/me'))).toHaveLength(1); // no retry without a session
  });

  it('keeps the session when the refresh fails for network/server reasons', async () => {
    const { client, tokens, onLost } = setup((call) =>
      call.url.endsWith('/auth/refresh') ? { status: 503 } : { status: 401 },
    );
    await expect(client.get('/me')).rejects.toBeInstanceOf(ApiError);
    expect(onLost).not.toHaveBeenCalled();
    expect(await tokens.get()).not.toBeNull();
  });

  it('does not log out when the retried request still returns 401 (e.g. wrong password)', async () => {
    const { client, tokens, onLost } = setup((call) => {
      if (call.url.endsWith('/auth/refresh')) {
        return { status: 200, body: { accessToken: 'A2', refreshToken: 'R2', user: {} } };
      }
      return { status: 401, body: { error: 'wrong password', code: 'unauthorized' } };
    });
    await expect(client.delete('/me', { password: 'x' })).rejects.toMatchObject({ status: 401 });
    expect(onLost).not.toHaveBeenCalled();
    expect((await tokens.get())?.accessToken).toBe('A2');
  });

  it('maps fetch failures to network ApiErrors and aborts on timeout', async () => {
    const tokens = memoryStore(null);
    const failing = createApiClient({
      baseUrl: 'http://api.test',
      tokens,
      fetchImpl: (async () => {
        throw new TypeError('Network request failed');
      }) as unknown as typeof fetch,
    });
    await expect(failing.get('/health', { auth: false })).rejects.toMatchObject({
      code: 'network',
      status: 0,
    });

    const hanging = createApiClient({
      baseUrl: 'http://api.test',
      tokens,
      timeoutMs: 20,
      fetchImpl: ((_u: string, init?: RequestInit) =>
        new Promise((_res, rej) => {
          init?.signal?.addEventListener('abort', () =>
            rej(Object.assign(new Error('aborted'), { name: 'AbortError' })),
          );
        })) as unknown as typeof fetch,
    });
    await expect(hanging.get('/slow', { auth: false })).rejects.toMatchObject({ code: 'timeout' });
  });
});
