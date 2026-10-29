import { ageFromIso, toIsoDate, validateBirthDate } from "./date";

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe("date helpers", () => {
  it("builds ISO dates only for real calendar days", () => {
    expect(toIsoDate({ day: "5", month: "3", year: "1994" })).toBe("1994-03-05");
    expect(toIsoDate({ day: "31", month: "2", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "29", month: "2", year: "2000" })).toBe("2000-02-29");
    expect(toIsoDate({ day: "29", month: "2", year: "2001" })).toBeNull();
    expect(toIsoDate({ day: "", month: "3", year: "1994" })).toBeNull();
    expect(toIsoDate({ day: "5", month: "3", year: "94" })).toBeNull();
  });

  it("computes whole-year ages around the birthday", () => {
    expect(ageFromIso("2008-06-15", NOW)).toBe(18);
    expect(ageFromIso("2008-06-16", NOW)).toBe(17);
    expect(ageFromIso("1990-01-01", NOW)).toBe(36);
  });

  it("refuses users under 18 and absurd ages", () => {
    expect(validateBirthDate({ day: "16", month: "6", year: "2008" }, NOW)).toHaveProperty("error");
    expect(validateBirthDate({ day: "1", month: "1", year: "1900" }, NOW)).toHaveProperty("error");
    expect(validateBirthDate({ day: "15", month: "6", year: "2008" }, NOW)).toEqual({
      iso: "2008-06-15"
    });
  });
});
