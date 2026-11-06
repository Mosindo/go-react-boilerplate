import { describe, expect, it } from "vitest";
import {
  ageFromISO,
  dayKey,
  displayBirthDate,
  formatClock,
  formatListTime,
  initials,
  parseBirthDate
} from "./format";

describe("parseBirthDate", () => {
  it("accepts valid dates", () => {
    expect(parseBirthDate("12/05/1994")).toBe("1994-05-12");
    expect(parseBirthDate(" 1.2.1990 ")).toBe("1990-02-01");
    expect(parseBirthDate("29/02/2000")).toBe("2000-02-29");
  });
  it("rejects impossible or malformed dates", () => {
    for (const bad of [
      "31/02/1990",
      "29/02/2001",
      "1990-05-12",
      "12/13/1990",
      "",
      "abc",
      "12/05/90"
    ]) {
      expect(parseBirthDate(bad)).toBeNull();
    }
  });
  it("round-trips for display", () => {
    expect(displayBirthDate("1994-05-12")).toBe("12/05/1994");
  });
});

describe("ageFromISO", () => {
  const now = new Date(2026, 5, 15);
  it("counts whole years", () => {
    expect(ageFromISO("2008-06-15", now)).toBe(18);
    expect(ageFromISO("2008-06-16", now)).toBe(17);
    expect(ageFromISO("1990-12-31", now)).toBe(35);
  });
});

describe("time helpers", () => {
  const now = new Date(2026, 5, 15, 18, 0);
  it("formats list times", () => {
    expect(formatListTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListTime(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListTime(new Date(2026, 4, 2, 10, 0).toISOString(), now)).toBe("02/05");
  });
  it("formats clock and day keys", () => {
    const iso = new Date(2026, 0, 3, 7, 8).toISOString();
    expect(formatClock(iso)).toBe("07:08");
    expect(dayKey(iso)).toBe("2026-01-03");
  });
  it("builds initials", () => {
    expect(initials("  léa")).toBe("L");
    expect(initials("")).toBe("?");
  });
});
