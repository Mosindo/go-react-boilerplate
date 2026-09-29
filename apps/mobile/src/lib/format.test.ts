import { describe, expect, it } from "vitest";
import {
  formatAge,
  formatAgeRange,
  formatCount,
  formatCounter,
  formatDistance,
  formatInterestedIn,
  formatKm,
  formatNameAge
} from "./format";

describe("format", () => {
  it("formats ages and names", () => {
    expect(formatAge(29)).toBe("29");
    expect(formatAge(null)).toBe("");
    expect(formatNameAge("Sam", 29)).toBe("Sam, 29");
    expect(formatNameAge("Sam", null)).toBe("Sam");
  });
  it("formats approximate distances", () => {
    expect(formatDistance(null)).toBe("");
    expect(formatDistance(1)).toBe("Less than 1 km away");
    expect(formatDistance(7)).toBe("7 km away");
    expect(formatDistance(45)).toBe("45 km away");
    expect(formatDistance(-3)).toBe("");
    expect(formatKm(50)).toBe("50 km");
  });
  it("formats ranges, counts and lists", () => {
    expect(formatAgeRange(25, 35)).toBe("25 to 35");
    expect(formatCount(0)).toBe("");
    expect(formatCount(7)).toBe("7");
    expect(formatCount(120)).toBe("99+");
    expect(formatCounter(12, 500)).toBe("12/500");
    expect(formatInterestedIn(["woman", "non_binary"])).toBe("Women, Non-binary people");
    expect(formatInterestedIn([])).toBe("Nobody yet");
  });
});
