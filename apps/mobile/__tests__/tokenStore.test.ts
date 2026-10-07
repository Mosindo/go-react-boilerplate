const mockStorage = new Map<string, string>();
let mockFailing = false;

jest.mock("expo-secure-store", () => ({
  setItemAsync: jest.fn(async (key: string, value: string) => {
    if (mockFailing) throw new Error("unavailable");
    mockStorage.set(key, value);
  }),
  getItemAsync: jest.fn(async (key: string) => {
    if (mockFailing) throw new Error("unavailable");
    return mockStorage.get(key) ?? null;
  }),
  deleteItemAsync: jest.fn(async (key: string) => {
    mockStorage.delete(key);
  })
}));

import { clearTokens, getAccessToken, getTokens, isAuthTokens, saveTokens } from "../src/store/tokenStore";

beforeEach(async () => {
  mockFailing = false;
  mockStorage.clear();
  await clearTokens();
});

describe("tokenStore", () => {
  it("persists and reads tokens", async () => {
    await saveTokens({ accessToken: "a", refreshToken: "r" });
    expect(await getTokens()).toEqual({ accessToken: "a", refreshToken: "r" });
    expect(await getAccessToken()).toBe("a");
    expect([...mockStorage.values()][0]).toContain("refreshToken");
  });

  it("clears memory and mockStorage", async () => {
    await saveTokens({ accessToken: "a", refreshToken: "r" });
    await clearTokens();
    expect(await getTokens()).toBeNull();
    expect(mockStorage.size).toBe(0);
  });

  it("keeps the session in memory when secure mockStorage fails", async () => {
    mockFailing = true;
    await saveTokens({ accessToken: "a", refreshToken: "r" });
    expect(await getAccessToken()).toBe("a");
  });

  it("restores from mockStorage on cold start and ignores corrupt data", async () => {
    mockStorage.set("ember.auth.tokens", JSON.stringify({ accessToken: "x", refreshToken: "y" }));
    expect(await getTokens()).toEqual({ accessToken: "x", refreshToken: "y" });
    await clearTokens();
    mockStorage.set("ember.auth.tokens", "{not json");
    expect(await getTokens()).toBeNull();
  });

  it("validates token shape", () => {
    expect(isAuthTokens({ accessToken: "a", refreshToken: "b" })).toBe(true);
    expect(isAuthTokens({ accessToken: "a", refreshToken: "" })).toBe(false);
    expect(isAuthTokens(null)).toBe(false);
  });
});
