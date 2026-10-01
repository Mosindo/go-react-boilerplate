import { ageOn, formatMessageTime, splitIsoDate, toIsoDate } from "../lib/dates";
import { distanceLabel, headline, truncate } from "../lib/format";
import { clamp, normalizeEmail, validateEmail, validateFirstName, validatePassword } from "../lib/validation";

describe("dates", () => {
  it("accepts only real calendar dates", () => {
    expect(toIsoDate("5", "3", "1994")).toBe("1994-03-05");
    expect(toIsoDate("29", "2", "2000")).toBe("2000-02-29");
    expect(toIsoDate("29", "2", "2001")).toBeNull();
    expect(toIsoDate("31", "4", "1990")).toBeNull();
    expect(toIsoDate("1", "13", "1990")).toBeNull();
    expect(toIsoDate("", "1", "1990")).toBeNull();
    expect(toIsoDate("1", "1", "90")).toBeNull();
    expect(toIsoDate("a", "1", "1990")).toBeNull();
  });

  it("computes age to the day", () => {
    const now = new Date(Date.UTC(2026, 5, 15));
    expect(ageOn("2008-06-15", now)).toBe(18);
    expect(ageOn("2008-06-16", now)).toBe(17);
    expect(ageOn("2000-12-31", now)).toBe(25);
  });

  it("splits an ISO date for the form", () => {
    expect(splitIsoDate("1994-03-05")).toEqual({ day: "5", month: "3", year: "1994" });
  });

  it("formats message times", () => {
    const now = new Date(2026, 5, 15, 12, 0, 0);
    expect(formatMessageTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toMatch(/09[:h]05/);
    expect(formatMessageTime(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("hier");
    expect(formatMessageTime(new Date(2026, 0, 3, 10, 0).toISOString(), now)).toBe("03/01");
  });
});

describe("validation", () => {
  it("normalises and validates emails", () => {
    expect(normalizeEmail("  Léa@Example.COM ")).toBe("léa@example.com");
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
    expect(validateEmail(" a@b.co ")).toBeNull();
  });

  it("enforces password bounds in bytes (bcrypt limit)", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long enough")).toBeNull();
    expect(validatePassword("é".repeat(37))).not.toBeNull(); // 74 bytes
    expect(validatePassword("é".repeat(36))).toBeNull(); // 72 bytes
  });

  it("validates names and clamps", () => {
    expect(validateFirstName("  ")).not.toBeNull();
    expect(validateFirstName("Zoé")).toBeNull();
    expect(validateFirstName("a".repeat(41))).not.toBeNull();
    expect(clamp(5, 10, 20)).toBe(10);
    expect(clamp(25, 10, 20)).toBe(20);
  });
});

describe("format", () => {
  it("shows approximate distances only", () => {
    expect(distanceLabel(null)).toBeNull();
    expect(distanceLabel(5)).toBe("à moins de 5 km");
    expect(distanceLabel(15)).toBe("à environ 15 km");
  });
  it("builds headlines and truncates", () => {
    expect(headline({ firstName: "Léa", age: 29 })).toBe("Léa, 29");
    expect(truncate("abcdef", 4)).toBe("abc…");
    expect(truncate("abc", 4)).toBe("abc");
  });
});
