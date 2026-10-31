import { ageFromBirthDate, parseBirthDate, validateEmail, validatePassword } from "../validation";

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe("parseBirthDate", () => {
  it("accepts someone turning 18 today", () => {
    const result = parseBirthDate("15", "06", "2008", NOW);
    expect(result).toEqual({ ok: true, iso: "2008-06-15" });
  });

  it("rejects someone turning 18 tomorrow", () => {
    expect(parseBirthDate("16", "06", "2008", NOW).ok).toBe(false);
  });

  it("rejects impossible dates", () => {
    expect(parseBirthDate("31", "02", "1990", NOW).ok).toBe(false);
    expect(parseBirthDate("aa", "02", "1990", NOW).ok).toBe(false);
    expect(parseBirthDate("10", "10", "90", NOW).ok).toBe(false);
  });

  it("rejects future and absurdly old dates", () => {
    expect(parseBirthDate("01", "01", "2030", NOW).ok).toBe(false);
    expect(parseBirthDate("01", "01", "1850", NOW).ok).toBe(false);
  });
});

describe("ageFromBirthDate", () => {
  it("counts full years only", () => {
    expect(ageFromBirthDate(new Date(1990, 5, 16), NOW)).toBe(35);
    expect(ageFromBirthDate(new Date(1990, 5, 15), NOW)).toBe(36);
  });
});

describe("credentials", () => {
  it("validates e-mail addresses", () => {
    expect(validateEmail("a@b.co")).toBeNull();
    expect(validateEmail("  A@B.CO ")).toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validateEmail("")).not.toBeNull();
  });

  it("enforces the 8..72 byte password window", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("é".repeat(40))).not.toBeNull(); // 80 bytes
  });
});
