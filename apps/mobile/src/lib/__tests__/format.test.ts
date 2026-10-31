import { formatDistance, formatTime } from "../format";

describe("formatDistance", () => {
  it("never shows a distance below 1 km", () => {
    expect(formatDistance(1)).toBe("à moins de 1 km");
    expect(formatDistance(15)).toBe("à 15 km");
    expect(formatDistance(undefined)).toBeNull();
  });
});

describe("formatTime", () => {
  const now = new Date(2026, 5, 15, 12, 0);
  it("returns an empty string for invalid dates", () => {
    expect(formatTime("garbage", now)).toBe("");
  });
  it("labels yesterday", () => {
    expect(formatTime(new Date(2026, 5, 14, 9, 0).toISOString(), now)).toBe("Hier");
  });
});
