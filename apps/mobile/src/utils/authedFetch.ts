export type Tokens = { accessToken: string; refreshToken: string };

export type TokenStore = {
  get(): Promise<Tokens | null>;
  set(tokens: Tokens): Promise<void>;
  clear(): Promise<void>;
};

export type AuthedFetchDeps = {
  baseUrl: string;
  fetchImpl: typeof fetch;
  store: TokenStore;
  onSessionLost: () => void;
};

/**
 * Builds a fetch wrapper that adds the bearer token and, on a 401, refreshes the
 * session exactly once (single-flight: concurrent 401s share one refresh call) and
 * retries the request one time. A failed refresh clears the session.
 */
export function createAuthedFetch(deps: AuthedFetchDeps): (path: string, init?: RequestInit) => Promise<Response> {
  let refreshing: Promise<Tokens | null> | null = null;

  async function refresh(failedAccessToken: string | null): Promise<Tokens | null> {
    if (refreshing) return refreshing;
    refreshing = (async () => {
      const current = await deps.store.get();
      // Another request may already have rotated the tokens.
      if (current && failedAccessToken && current.accessToken !== failedAccessToken) return current;
      if (!current?.refreshToken) return null;
      try {
        const res = await deps.fetchImpl(`${deps.baseUrl}/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ refreshToken: current.refreshToken }),
        });
        if (!res.ok) {
          if (res.status === 401 || res.status === 400) {
            await deps.store.clear();
            deps.onSessionLost();
          }
          return null;
        }
        const data = (await res.json()) as Tokens;
        const next = { accessToken: data.accessToken, refreshToken: data.refreshToken };
        await deps.store.set(next);
        return next;
      } catch {
        return null; // network failure: keep the session, caller sees the error
      }
    })().finally(() => {
      refreshing = null;
    });
    return refreshing;
  }

  async function send(path: string, init: RequestInit | undefined, token: string | null): Promise<Response> {
    const headers = new Headers(init?.headers);
    headers.set("Accept", "application/json");
    if (token) headers.set("Authorization", `Bearer ${token}`);
    return deps.fetchImpl(`${deps.baseUrl}${path}`, { ...init, headers });
  }

  return async (path, init) => {
    const tokens = await deps.store.get();
    const res = await send(path, init, tokens?.accessToken ?? null);
    if (res.status !== 401 || !tokens) return res;
    const next = await refresh(tokens.accessToken);
    if (!next) return res;
    return send(path, init, next.accessToken);
  };
}
