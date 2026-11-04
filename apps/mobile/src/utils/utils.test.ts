import { describe, expect, it, vi } from "vitest";
import { ageOn, isoToFrench, maskBirthDate, parseBirthDate } from "./dates";
import { cleanMessage, validateEmail, validatePassword, validatePreferences } from "./validation";
import { formatHour, formatListTime, relativeTime } from "./time";
import { resolveAssetUrl, resolveBaseUrl, websocketUrl } from "./url";
import { backoffDelay } from "./backoff";
import { mergeDeck, shouldPrefetch } from "./deck";
import { createAuthedFetch, type TokenStore, type Tokens } from "./authedFetch";

const today = new Date(Date.UTC(2026, 5, 15));

describe("birth date", () => {
  it("accepts adults and returns ISO + age", () => {
    expect(parseBirthDate("15/06/2008", today)).toEqual({ ok: true, iso: "2008-06-15", age: 18 });
  });
  it("rejects minors by one day", () => {
    const r = parseBirthDate("16/06/2008", today);
    expect(r.ok).toBe(false);
  });
  it("rejects malformed and impossible dates", () => {
    expect(parseBirthDate("1990-01-01", today).ok).toBe(false);
    expect(parseBirthDate("31/02/1990", today).ok).toBe(false);
    expect(parseBirthDate("01/01/1800", today).ok).toBe(false);
    expect(parseBirthDate("01/01/2999", today).ok).toBe(false);
  });
  it("computes age and masks input", () => {
    expect(ageOn(1990, 6, 16, today)).toBe(35);
    expect(maskBirthDate("15062008")).toBe("15/06/2008");
    expect(maskBirthDate("1506")).toBe("15/06");
    expect(isoToFrench("2008-06-15")).toBe("15/06/2008");
  });
});

describe("validation", () => {
  it("validates preferences", () => {
    const ok = { interestedIn: ["man"], minAge: 20, maxAge: 30, maxDistanceKm: 50 };
    expect(validatePreferences(ok)).toBeNull();
    expect(validatePreferences({ ...ok, interestedIn: [] })).not.toBeNull();
    expect(validatePreferences({ ...ok, minAge: 17 })).not.toBeNull();
    expect(validatePreferences({ ...ok, minAge: 40 })).not.toBeNull();
    expect(validatePreferences({ ...ok, maxDistanceKm: 0 })).not.toBeNull();
    expect(validatePreferences({ ...ok, maxDistanceKm: 501 })).not.toBeNull();
  });
  it("validates credentials", () => {
    expect(validateEmail("a@b.co")).toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validatePassword("Passw0rdTest")).toBeNull();
    expect(validatePassword("short1")).not.toBeNull();
    expect(validatePassword("onlyletters")).not.toBeNull();
  });
  it("cleans messages", () => {
    expect(cleanMessage("  salut \u0000 ")).toEqual({ ok: true, body: "salut" });
    expect(cleanMessage("   ").ok).toBe(false);
    expect(cleanMessage("x".repeat(2001)).ok).toBe(false);
    expect(cleanMessage("x".repeat(2000)).ok).toBe(true);
  });
});

describe("time", () => {
  const now = new Date(2026, 5, 15, 12, 0, 0);
  it("formats hours and list labels", () => {
    expect(formatHour(new Date(2026, 5, 15, 9, 5).toISOString())).toBe("09:05");
    expect(formatListTime(new Date(2026, 5, 15, 8, 30).toISOString(), now)).toBe("08:30");
    expect(formatListTime(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListTime(new Date(2026, 4, 1, 10, 0).toISOString(), now)).toBe("01/05");
  });
  it("formats relative times", () => {
    expect(relativeTime(new Date(now.getTime() - 5_000).toISOString(), now)).toBe("à l'instant");
    expect(relativeTime(new Date(now.getTime() - 5 * 60_000).toISOString(), now)).toBe("il y a 5 min");
    expect(relativeTime(new Date(now.getTime() - 3 * 3_600_000).toISOString(), now)).toBe("il y a 3 h");
    expect(relativeTime(new Date(now.getTime() - 2 * 86_400_000).toISOString(), now)).toBe("il y a 2 j");
  });
});

describe("urls", () => {
  it("resolves base url", () => {
    expect(resolveBaseUrl({ explicit: " http://x:1/ ", isDev: true, os: "ios" })).toBe("http://x:1");
    expect(resolveBaseUrl({ isDev: true, os: "android" })).toBe("http://10.0.2.2:18080");
    expect(resolveBaseUrl({ isDev: true, os: "ios" })).toBe("http://localhost:18080");
    expect(resolveBaseUrl({ isDev: false, os: "ios" })).toBe("");
  });
  it("resolves assets and websocket", () => {
    expect(resolveAssetUrl("http://h", "/photos/1/file?exp=2")).toBe("http://h/photos/1/file?exp=2");
    expect(resolveAssetUrl("http://h", "https://cdn/x.jpg")).toBe("https://cdn/x.jpg");
    expect(websocketUrl("https://api.example.com")).toBe("wss://api.example.com/ws");
    expect(websocketUrl("http://h:1")).toBe("ws://h:1/ws");
  });
});

describe("backoff", () => {
  it("grows exponentially, is jittered and capped", () => {
    expect(backoffDelay(0, () => 1)).toBe(1000);
    expect(backoffDelay(0, () => 0)).toBe(500);
    expect(backoffDelay(3, () => 1)).toBe(8000);
    expect(backoffDelay(20, () => 1)).toBe(30_000);
    expect(backoffDelay(20, () => 0)).toBe(15_000);
  });
});

describe("deck", () => {
  it("dedupes and skips judged profiles", () => {
    const merged = mergeDeck([{ userId: "a" }], [{ userId: "a" }, { userId: "b" }, { userId: "c" }, { userId: "b" }], new Set(["c"]));
    expect(merged.map((p) => p.userId)).toEqual(["a", "b"]);
  });
  it("decides when to prefetch", () => {
    expect(shouldPrefetch(3, false, false)).toBe(true);
    expect(shouldPrefetch(4, false, false)).toBe(false);
    expect(shouldPrefetch(1, true, false)).toBe(false);
    expect(shouldPrefetch(0, false, true)).toBe(false);
  });
});

function memoryStore(initial: Tokens | null): TokenStore & { value: Tokens | null } {
  const s = {
    value: initial,
    get: async () => s.value,
    set: async (t: Tokens) => {
      s.value = t;
    },
    clear: async () => {
      s.value = null;
    },
  };
  return s;
}

const json = (status: number, body: unknown): Response => new Response(JSON.stringify(body), { status });

describe("authed fetch", () => {
  it("refreshes once for concurrent 401s and retries", async () => {
    const store = memoryStore({ accessToken: "old", refreshToken: "r1" });
    let refreshCalls = 0;
    const fetchImpl = vi.fn(async (url: string | URL | Request, init?: RequestInit) => {
      const u = String(url);
      if (u.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return json(200, { accessToken: "new", refreshToken: "r2" });
      }
      const auth = new Headers(init?.headers).get("Authorization");
      return auth === "Bearer new" ? json(200, { ok: true }) : json(401, { error: "x" });
    }) as unknown as typeof fetch;
    const f = createAuthedFetch({ baseUrl: "http://h", fetchImpl, store, onSessionLost: () => undefined });
    const results = await Promise.all([f("/a"), f("/b"), f("/c")]);
    expect(results.map((r) => r.status)).toEqual([200, 200, 200]);
    expect(refreshCalls).toBe(1);
    expect(store.value).toEqual({ accessToken: "new", refreshToken: "r2" });
  });
  it("clears the session when refresh is rejected", async () => {
    const store = memoryStore({ accessToken: "old", refreshToken: "r1" });
    const lost = vi.fn();
    const fetchImpl = vi.fn(async () => json(401, { error: "no" })) as unknown as typeof fetch;
    const f = createAuthedFetch({ baseUrl: "http://h", fetchImpl, store, onSessionLost: lost });
    const res = await f("/a");
    expect(res.status).toBe(401);
    expect(store.value).toBeNull();
    expect(lost).toHaveBeenCalledOnce();
  });
  it("keeps the session on network errors", async () => {
    const store = memoryStore({ accessToken: "old", refreshToken: "r1" });
    const lost = vi.fn();
    const fetchImpl = vi.fn(async (url: string | URL | Request) => {
      if (String(url).endsWith("/auth/refresh")) throw new TypeError("offline");
      return json(401, {});
    }) as unknown as typeof fetch;
    const f = createAuthedFetch({ baseUrl: "http://h", fetchImpl, store, onSessionLost: lost });
    expect((await f("/a")).status).toBe(401);
    expect(store.value).not.toBeNull();
    expect(lost).not.toHaveBeenCalled();
  });
});

import { flattenPages, isReadByPeer, latestOwnMessageId } from "./messages";

describe("messages", () => {
  it("flattens pages newest-first", () => {
    const pages = [{ messages: [{ id: "4" }, { id: "5" }] }, { messages: [{ id: "1" }, { id: "2" }, { id: "3" }] }];
    expect(flattenPages(pages).map((m) => m.id)).toEqual(["5", "4", "3", "2", "1"]);
  });
  it("compares read pointers with big integers", () => {
    expect(isReadByPeer("10", "10")).toBe(true);
    expect(isReadByPeer("11", "10")).toBe(false);
    expect(isReadByPeer("9007199254740993", "9007199254740992")).toBe(false);
    expect(isReadByPeer("tmp-1", "10")).toBe(false);
  });
  it("finds the latest own message", () => {
    const list = [{ id: "3", senderId: "b" }, { id: "2", senderId: "a" }, { id: "1", senderId: "a" }];
    expect(latestOwnMessageId(list, "a")).toBe("2");
    expect(latestOwnMessageId(list, "z")).toBeNull();
  });
});
