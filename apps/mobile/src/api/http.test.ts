import { createHttpClient } from './http';
import type { FetchLike, TokenStorage, Tokens } from './http';

function memoryStorage(initial: Tokens | null): TokenStorage & { current: Tokens | null } {
  const store = {
    current: initial,
    load: async () => store.current,
    save: async (t: Tokens) => {
      store.current = t;
    },
    clear: async () => {
      store.current = null;
    },
  };
  return store;
}

const json = (status: number, body: unknown): Response =>
  new Response(body === undefined ? null : JSON.stringify(body), { status });

interface Call {
  url: string;
  auth: string | undefined;
  method: string | undefined;
  body: string | undefined;
}

function setup(handler: (call: Call) => Response | Promise<Response>, tokens: Tokens | null) {
  const calls: Call[] = [];
  const fetchImpl: FetchLike = async (url, init) => {
    const headers = init?.headers as Record<string, string> | undefined;
    const call: Call = {
      url,
      auth: headers?.Authorization,
      method: init?.method,
      body: typeof init?.body === 'string' ? init.body : undefined,
    };
    calls.push(call);
    return handler(call);
  };
  const storage = memoryStorage(tokens);
  const client = createHttpClient({ baseUrl: 'http://api', storage, fetchImpl });
  return { client, calls, storage };
}

const OLD: Tokens = { accessToken: 'old-access', refreshToken: 'old-refresh' };
const refreshed = (n = 1) => ({
  accessToken: `new-access-${n}`,
  refreshToken: `new-refresh-${n}`,
  user: { id: 'u', email: 'a@b.fr', createdAt: 'x', profileComplete: true },
});

describe('http client', () => {
  it('sends the bearer token, JSON body and query string', async () => {
    const { client, calls } = setup(() => json(200, { ok: true }), OLD);
    const res = await client.request<{ ok: boolean }>('POST', '/x', {
      body: { a: 1 },
      query: { limit: 10, cursor: undefined, q: 'a b' },
    });
    expect(res).toEqual({ ok: true });
    expect(calls[0].url).toBe('http://api/x?limit=10&q=a%20b');
    expect(calls[0].auth).toBe('Bearer old-access');
    expect(calls[0].body).toBe('{"a":1}');
  });

  it('does not attach a token to public endpoints and handles 204', async () => {
    const { client, calls } = setup(() => json(204, undefined), OLD);
    await expect(client.request('POST', '/auth/login', { auth: false })).resolves.toBeUndefined();
    expect(calls[0].auth).toBeUndefined();
  });

  it('refreshes once on 401, rotates and persists tokens, then retries', async () => {
    const { client, calls, storage } = setup((call) => {
      if (call.url.endsWith('/auth/refresh')) return json(200, refreshed());
      return call.auth === 'Bearer new-access-1'
        ? json(200, { ok: true })
        : json(401, { error: 'x', code: 'unauthorized' });
    }, OLD);
    await expect(client.request('GET', '/me')).resolves.toEqual({ ok: true });
    expect(calls.map((c) => c.url)).toEqual([
      'http://api/me',
      'http://api/auth/refresh',
      'http://api/me',
    ]);
    expect(calls[1].body).toBe('{"refreshToken":"old-refresh"}');
    expect(storage.current).toEqual({ accessToken: 'new-access-1', refreshToken: 'new-refresh-1' });
  });

  it('single-flights concurrent 401s into one refresh call', async () => {
    let refreshCalls = 0;
    const { client, calls } = setup(async (call) => {
      if (call.url.endsWith('/auth/refresh')) {
        refreshCalls += 1;
        await new Promise((r) => setTimeout(r, 10));
        return json(200, refreshed());
      }
      return call.auth === 'Bearer new-access-1'
        ? json(200, { path: call.url })
        : json(401, { error: 'x', code: 'unauthorized' });
    }, OLD);
    const results = await Promise.all([
      client.request('GET', '/a'),
      client.request('GET', '/b'),
      client.request('GET', '/c'),
    ]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
    expect(calls.filter((c) => c.auth === 'Bearer new-access-1')).toHaveLength(3);
  });

  it('a request that failed with a stale token reuses the already rotated one', async () => {
    let refreshCalls = 0;
    let release: () => void = () => undefined;
    const gate = new Promise<void>((r) => (release = r));
    const { client } = setup(async (call) => {
      if (call.url.endsWith('/auth/refresh')) {
        refreshCalls += 1;
        return json(200, refreshed());
      }
      if (call.url.endsWith('/slow') && call.auth === 'Bearer old-access') {
        await gate; // still in flight while another request refreshes
        return json(401, { error: 'x', code: 'unauthorized' });
      }
      return call.auth === 'Bearer new-access-1'
        ? json(200, { ok: true })
        : json(401, { error: 'x', code: 'unauthorized' });
    }, OLD);
    const slow = client.request('GET', '/slow');
    await client.request('GET', '/fast');
    release();
    await expect(slow).resolves.toEqual({ ok: true });
    expect(refreshCalls).toBe(1);
  });

  it("'optional' auth attaches the token but never refreshes (logout)", async () => {
    const { client, calls } = setup(() => json(401, { error: 'x', code: 'unauthorized' }), OLD);
    await expect(
      client.request('POST', '/auth/logout', { auth: 'optional' }),
    ).rejects.toMatchObject({
      status: 401,
    });
    expect(calls).toHaveLength(1);
    expect(calls[0].auth).toBe('Bearer old-access');
  });

  it('retries only once: a second 401 surfaces as an error', async () => {
    const { client, calls } = setup((call) => {
      if (call.url.endsWith('/auth/refresh')) return json(200, refreshed());
      return json(401, { error: 'nope', code: 'unauthorized' });
    }, OLD);
    await expect(client.request('GET', '/me')).rejects.toMatchObject({
      status: 401,
      code: 'unauthorized',
    });
    expect(calls.filter((c) => c.url.endsWith('/me'))).toHaveLength(2);
    expect(calls.filter((c) => c.url.endsWith('/auth/refresh'))).toHaveLength(1);
  });

  it('logs out (clears tokens, notifies) when the refresh is rejected', async () => {
    const { client, storage } = setup(
      (call) =>
        call.url.endsWith('/auth/refresh')
          ? json(401, { error: 'revoked', code: 'unauthorized' })
          : json(401, { error: 'x', code: 'unauthorized' }),
      OLD,
    );
    const onExpired = jest.fn();
    client.onSessionExpired(onExpired);
    await expect(client.request('GET', '/me')).rejects.toMatchObject({ status: 401 });
    expect(onExpired).toHaveBeenCalledTimes(1);
    expect(storage.current).toBeNull();
    expect(client.getRefreshToken()).toBeNull();
  });

  it('keeps the session when the refresh fails because of the network', async () => {
    const { client, storage } = setup((call) => {
      if (call.url.endsWith('/auth/refresh')) throw new TypeError('Network request failed');
      return json(401, { error: 'x', code: 'unauthorized' });
    }, OLD);
    const onExpired = jest.fn();
    client.onSessionExpired(onExpired);
    await expect(client.request('GET', '/me')).rejects.toMatchObject({ code: 'network' });
    expect(onExpired).not.toHaveBeenCalled();
    expect(storage.current).toEqual(OLD);
  });

  it('maps fetch failures to a friendly network error', async () => {
    const { client } = setup(() => {
      throw new TypeError('Failed to fetch');
    }, OLD);
    await expect(client.request('GET', '/me')).rejects.toMatchObject({
      code: 'network',
      status: 0,
    });
  });

  it('surfaces documented error bodies as ApiError', async () => {
    const { client } = setup(() => json(409, { error: 'taken', code: 'email_taken' }), null);
    await expect(client.request('POST', '/auth/register', { auth: false })).rejects.toMatchObject({
      status: 409,
      code: 'email_taken',
    });
  });

  it('setSession / clearSession drive the stored tokens', async () => {
    const { client, storage } = setup(() => json(200, {}), null);
    await client.setSession(OLD);
    expect(storage.current).toEqual(OLD);
    expect(client.getRefreshToken()).toBe('old-refresh');
    await client.clearSession();
    expect(storage.current).toBeNull();
  });
});
