import { formatDistance, formatShortTime } from "../lib/format";

describe("formatShortTime", () => {
  const now = new Date(2026, 5, 15, 18, 0, 0);

  it("shows the clock for today", () => {
    expect(formatShortTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toMatch(/09[:h]05/);
  });
  it("says hier for yesterday", () => {
    expect(formatShortTime(new Date(2026, 5, 14, 23, 59).toISOString(), now)).toBe("hier");
  });
  it("falls back to dd/mm for old dates", () => {
    expect(formatShortTime(new Date(2026, 2, 3, 12).toISOString(), now)).toBe("03/03");
  });
  it("returns empty string for garbage", () => {
    expect(formatShortTime("not a date", now)).toBe("");
  });
});

describe("formatDistance", () => {
  it("never claims more precision than the server gives", () => {
    expect(formatDistance(undefined)).toBeNull();
    expect(formatDistance(5)).toBe("À moins de 5 km");
    expect(formatDistance(40)).toBe("À environ 40 km");
  });
});
