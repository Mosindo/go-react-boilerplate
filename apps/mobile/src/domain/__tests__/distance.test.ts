import { formatDistance } from "../distance";

describe("formatDistance", () => {
  it("returns null when the distance is hidden or unknown", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(undefined)).toBeNull();
    expect(formatDistance(Number.NaN)).toBeNull();
  });

  it("describes the first bucket as within 5 km", () => {
    expect(formatDistance(5)).toBe("Within 5 km");
    expect(formatDistance(3)).toBe("Within 5 km");
  });

  it("describes larger buckets as approximate", () => {
    expect(formatDistance(10)).toBe("About 10 km away");
    expect(formatDistance(120)).toBe("About 120 km away");
  });
});
