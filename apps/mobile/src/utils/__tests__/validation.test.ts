import { ageOn, formatBirthDateInput, isValidEmail, isoToDisplayDate, parseBirthDate, passwordProblem } from "../validation";

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts a valid adult date and returns the API format", () => {
    const result = parseBirthDate("12/03/1990", NOW);
    expect(result).toEqual({ ok: true, iso: "1990-03-12", age: 36 });
  });

  it("is exactly 18 on the birthday and 17 the day before", () => {
    expect(parseBirthDate("15/06/2008", NOW).ok).toBe(true);
    const tomorrow = parseBirthDate("16/06/2008", NOW);
    expect(tomorrow.ok).toBe(false);
    if (!tomorrow.ok) expect(tomorrow.error).toMatch(/18 ans/);
  });

  it.each(["", "1990-03-12", "12-03-1990", "1/3/1990", "ab/cd/efgh"])("rejects malformed input %p", (input) => {
    expect(parseBirthDate(input, NOW).ok).toBe(false);
  });

  it.each(["31/02/1990", "00/10/1990", "15/13/1990", "29/02/2001"])("rejects impossible date %p", (input) => {
    const result = parseBirthDate(input, NOW);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error).toMatch(/n'existe pas/);
  });

  it("accepts a real leap day and rejects implausible ages", () => {
    expect(parseBirthDate("29/02/2000", NOW).ok).toBe(true);
    expect(parseBirthDate("01/01/1900", NOW).ok).toBe(false);
  });
});

describe("ageOn", () => {
  it("does not count the current year before the birthday", () => {
    expect(ageOn(new Date(2000, 11, 31), new Date(2026, 0, 1))).toBe(25);
    expect(ageOn(new Date(2000, 0, 1), new Date(2026, 0, 1))).toBe(26);
  });
});

describe("formatBirthDateInput", () => {
  it("inserts slashes while typing and ignores non digits", () => {
    expect(formatBirthDateInput("1")).toBe("1");
    expect(formatBirthDateInput("120")).toBe("12/0");
    expect(formatBirthDateInput("12031990")).toBe("12/03/1990");
    expect(formatBirthDateInput("12/03/1990999")).toBe("12/03/1990");
    expect(formatBirthDateInput("a1b2")).toBe("12");
  });
});

describe("credentials validation", () => {
  it("validates emails", () => {
    expect(isValidEmail(" jane@example.com ")).toBe(true);
    expect(isValidEmail("jane@example")).toBe(false);
    expect(isValidEmail("jane example@x.com")).toBe(false);
    expect(isValidEmail("")).toBe(false);
  });

  it("mirrors the API password rule (8-72)", () => {
    expect(passwordProblem("short")).not.toBeNull();
    expect(passwordProblem("longenough")).toBeNull();
    expect(passwordProblem("x".repeat(73))).not.toBeNull();
    expect(passwordProblem("x".repeat(72))).toBeNull();
  });

  it("converts API dates for display", () => {
    expect(isoToDisplayDate("1990-03-12")).toBe("12/03/1990");
  });
});
