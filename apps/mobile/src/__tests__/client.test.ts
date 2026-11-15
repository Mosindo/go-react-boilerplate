import { api, API_BASE_URL, ApiError, onSessionChange } from "../api/client";
import { errorMessage } from "../lib/errors";
import { currentTokens, saveTokens } from "../session/tokenStore";

// jest.mock calls are hoisted above imports by babel-jest.
jest.mock("expo-secure-store", () => {
  const store = new Map<string, string>();
  return {
    getItemAsync: jest.fn(async (k: string) => store.get(k) ?? null),
    setItemAsync: jest.fn(async (k: string, v: string) => void store.set(k, v)),
    deleteItemAsync: jest.fn(async (k: string) => void store.delete(k))
  };
});

type Call = { url: string; init: RequestInit };

function respond(status: number, body: unknown) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(body === undefined ? "" : JSON.stringify(body))
  });
}

describe("api client", () => {
  let calls: Call[];

  beforeEach(async () => {
    calls = [];
    await saveTokens({ accessToken: "old-access", refreshToken: "refresh-1" });
  });

  it("sends the bearer token and parses JSON", async () => {
    global.fetch = jest.fn((url: string, init: RequestInit) => {
      calls.push({ url, init });
      return respond(200, { ok: true });
    }) as unknown as typeof fetch;
    await expect(api("/me")).resolves.toEqual({ ok: true });
    expect((calls[0].init.headers as Record<string, string>).Authorization).toBe("Bearer old-access");
  });

  it("refreshes once on 401 and retries, sharing the refresh between concurrent calls", async () => {
    let refreshes = 0;
    global.fetch = jest.fn((url: string, init: RequestInit) => {
      calls.push({ url, init });
      const auth = (init.headers as Record<string, string>).Authorization;
      if (url.endsWith("/auth/refresh")) {
        refreshes += 1;
        return respond(200, { accessToken: "new-access", refreshToken: "refresh-2", user: { id: "u1" } });
      }
      return auth === "Bearer new-access" ? respond(200, { ok: url }) : respond(401, { error: "invalid token", code: "unauthorized" });
    }) as unknown as typeof fetch;

    const seen: unknown[] = [];
    const off = onSessionChange((s) => seen.push(s));
    const [a, b] = await Promise.all([api("/a"), api("/b")]);
    off();

    expect(a).toEqual({ ok: `${API_BASE_URL}/a` });
    expect(b).toEqual({ ok: `${API_BASE_URL}/b` });
    expect(refreshes).toBe(1);
    expect(currentTokens()).toEqual({ accessToken: "new-access", refreshToken: "refresh-2" });
    expect(seen).toHaveLength(1);
  });

  it("signs out when the refresh token is rejected", async () => {
    global.fetch = jest.fn((url: string) =>
      url.endsWith("/auth/refresh")
        ? respond(401, { error: "expired", code: "invalid_refresh_token" })
        : respond(401, { code: "unauthorized" })
    ) as unknown as typeof fetch;
    const seen: unknown[] = [];
    const off = onSessionChange((s) => seen.push(s));
    await expect(api("/me")).rejects.toBeInstanceOf(ApiError);
    off();
    expect(seen).toEqual([null]);
    expect(currentTokens()).toBeNull();
  });

  it("maps errors to French copy", async () => {
    global.fetch = jest.fn(() => respond(409, { error: "already", code: "already_swiped" })) as unknown as typeof fetch;
    const error = await api("/swipes", { method: "POST", body: {} }).catch((e: unknown) => e);
    expect(errorMessage(error)).toBe("Vous avez déjà répondu à ce profil.");

    global.fetch = jest.fn(() => Promise.reject(new TypeError("Network request failed"))) as unknown as typeof fetch;
    const network = await api("/me").catch((e: unknown) => e);
    expect(network).toBeInstanceOf(ApiError);
    expect(errorMessage(network)).toMatch(/Connexion impossible/);
  });
});
