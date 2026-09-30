import { ageFromIso, formatDistance, formatRelative, maskDateInput, parseFrenchDate, passwordProblem } from "./format";

describe("format helpers", () => {
  it("parses valid French dates and rejects impossible ones", () => {
    expect(parseFrenchDate("01/02/1995")).toBe("1995-02-01");
    expect(parseFrenchDate("31/02/1995")).toBeNull();
    expect(parseFrenchDate("1995-02-01")).toBeNull();
  });

  it("masks date input progressively", () => {
    expect(maskDateInput("0")).toBe("0");
    expect(maskDateInput("0102")).toBe("01/02");
    expect(maskDateInput("01021995extra")).toBe("01/02/1995");
  });

  it("computes age around birthdays", () => {
    const now = new Date(2026, 5, 15);
    expect(ageFromIso("2008-06-15", now)).toBe(18);
    expect(ageFromIso("2008-06-16", now)).toBe(17);
  });

  it("formats approximate distances", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(2)).toBe("À moins de 2 km");
    expect(formatDistance(15)).toBe("À 15 km");
  });

  it("formats relative times", () => {
    const now = new Date("2026-01-01T12:00:00Z");
    expect(formatRelative("2026-01-01T11:59:30Z", now)).toBe("À l'instant");
    expect(formatRelative("2026-01-01T11:30:00Z", now)).toBe("Il y a 30 min");
    expect(formatRelative("2026-01-01T09:00:00Z", now)).toBe("Il y a 3 h");
  });

  it("validates passwords like the API", () => {
    expect(passwordProblem("short1")).not.toBeNull();
    expect(passwordProblem("onlyletters")).not.toBeNull();
    expect(passwordProblem("Password1")).toBeNull();
  });
});
