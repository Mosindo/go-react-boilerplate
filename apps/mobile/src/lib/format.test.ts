import { ageOn, formatDayLabel, formatDistance, formatListTime, isAdult, parseBirthDateInput, sameDay } from "./format";

describe("parseBirthDateInput", () => {
  it("accepts real dates in several separators", () => {
    expect(parseBirthDateInput("14/03/1995")).toBe("1995-03-14");
    expect(parseBirthDateInput(" 1.2.1990 ")).toBe("1990-02-01");
    expect(parseBirthDateInput("29/02/2000")).toBe("2000-02-29");
  });

  it("rejects impossible or malformed dates", () => {
    for (const bad of ["31/02/1995", "29/02/2001", "1995-03-14", "14/13/1995", "", "abc", "14/03/95"]) {
      expect(parseBirthDateInput(bad)).toBeNull();
    }
  });
});

describe("age rules", () => {
  const today = new Date(2026, 5, 15);

  it("computes whole years", () => {
    expect(ageOn("2008-06-15", today)).toBe(18);
    expect(ageOn("2008-06-16", today)).toBe(17);
  });

  it("enforces majority", () => {
    expect(isAdult("2008-06-15", today)).toBe(true);
    expect(isAdult("2008-06-16", today)).toBe(false);
  });
});

describe("display helpers", () => {
  it("formats the coarse distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(5)).toBe("À moins de 5 km");
    expect(formatDistance(15)).toBe("À 15 km");
  });

  it("formats list times relative to today", () => {
    const now = new Date(2026, 5, 15, 12, 0);
    expect(formatListTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListTime(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListTime(new Date(2026, 2, 1, 10, 0).toISOString(), now)).toBe("01/03");
    expect(formatDayLabel(new Date(2026, 5, 15).toISOString(), now)).toBe("Aujourd'hui");
  });

  it("compares calendar days", () => {
    expect(sameDay(new Date(2026, 5, 15, 1).toISOString(), new Date(2026, 5, 15, 23).toISOString())).toBe(true);
    expect(sameDay(new Date(2026, 5, 15, 23).toISOString(), new Date(2026, 5, 16, 1).toISOString())).toBe(false);
  });
});
