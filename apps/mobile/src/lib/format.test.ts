import { describe, expect, it } from "vitest";
import { formatDistance, formatRelative } from "./format";

describe("formatRelative", () => {
  const now = new Date(2026, 5, 15, 12, 0, 0);
  it("words recent times", () => {
    expect(formatRelative(new Date(2026, 5, 15, 11, 59, 40).toISOString(), now)).toBe("À l'instant");
    expect(formatRelative(new Date(2026, 5, 15, 11, 30, 0).toISOString(), now)).toBe("il y a 30 min");
  });
  it("uses the clock later the same day, then yesterday, then the date", () => {
    expect(formatRelative(new Date(2026, 5, 15, 8, 5, 0).toISOString(), now)).toBe("08:05");
    expect(formatRelative(new Date(2026, 5, 14, 23, 0, 0).toISOString(), now)).toBe("Hier");
    expect(formatRelative(new Date(2026, 2, 3, 10, 0, 0).toISOString(), now)).toBe("3 mars");
    expect(formatRelative(new Date(2025, 2, 3, 10, 0, 0).toISOString(), now)).toBe("3 mars 2025");
  });
  it("tolerates garbage", () => {
    expect(formatRelative("nope", now)).toBe("");
  });
});

describe("formatDistance", () => {
  it("words coarse distances", () => {
    expect(formatDistance(1)).toBe("à moins de 1 km");
    expect(formatDistance(12)).toBe("à 12 km");
    expect(formatDistance(null)).toBeNull();
  });
});
