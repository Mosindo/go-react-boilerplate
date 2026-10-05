import { describe, expect, it } from "vitest";
import { TokenStore, type TokenBackend } from "./tokenStore";

function backend(initial: string | null = null, failing = false): TokenBackend & { value: string | null } {
  const b = {
    value: initial,
    get: async () => {
      if (failing) throw new Error("locked");
      return b.value;
    },
    set: async (v: string) => {
      if (failing) throw new Error("locked");
      b.value = v;
    },
    remove: async () => {
      if (failing) throw new Error("locked");
      b.value = null;
    }
  };
  return b;
}

describe("TokenStore", () => {
  it("round-trips tokens through the backend", async () => {
    const b = backend();
    await new TokenStore(b).save({ accessToken: "a", refreshToken: "r" });
    expect(await new TokenStore(b).load()).toEqual({ accessToken: "a", refreshToken: "r" });
  });

  it("treats corrupted or incomplete entries as signed out", async () => {
    for (const raw of ["not json", "{}", JSON.stringify({ accessToken: "a" }), JSON.stringify({ accessToken: "", refreshToken: "r" })]) {
      expect(await new TokenStore(backend(raw)).load()).toBeNull();
    }
  });

  it("keeps the session in memory when secure storage fails", async () => {
    const store = new TokenStore(backend(null, true));
    await store.save({ accessToken: "a", refreshToken: "r" });
    expect(await store.load()).toEqual({ accessToken: "a", refreshToken: "r" });
    await store.clear();
    expect(await store.load()).toBeNull();
  });

  it("clear() wipes storage", async () => {
    const b = backend(JSON.stringify({ accessToken: "a", refreshToken: "r" }));
    const store = new TokenStore(b);
    await store.load();
    await store.clear();
    expect(b.value).toBeNull();
    expect(await store.load()).toBeNull();
  });
});
