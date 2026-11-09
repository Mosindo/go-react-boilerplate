import { isExpiring, tokenExpiry } from "./jwt";

function fakeToken(exp: number): string {
  const payload = btoa(JSON.stringify({ exp })).replace(/=+$/, "").replace(/\+/g, "-").replace(/\//g, "_");
  return `header.${payload}.sig`;
}

describe("jwt helpers", () => {
  it("reads exp", () => {
    expect(tokenExpiry(fakeToken(1_800_000_000))).toBe(1_800_000_000);
  });
  it("returns null for garbage", () => {
    expect(tokenExpiry("garbage")).toBeNull();
    expect(tokenExpiry("a.%%%.c")).toBeNull();
  });
  it("detects tokens about to expire", () => {
    const now = 1_000_000_000_000;
    expect(isExpiring(fakeToken(now / 1000 + 10), now)).toBe(true);
    expect(isExpiring(fakeToken(now / 1000 + 600), now)).toBe(false);
    expect(isExpiring("garbage", now)).toBe(true);
  });
});
