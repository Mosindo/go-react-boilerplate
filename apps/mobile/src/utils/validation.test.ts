import { ageOn, digitsOnly, parseBirthdate, splitBirthdate, validateEmail, validateFirstName, validatePassword } from "./validation";

const NOW = new Date(2026, 2, 15); // 15 March 2026

describe("validateEmail", () => {
  it("accepts normal addresses and rejects malformed ones", () => {
    expect(validateEmail(" a@b.fr ")).toBeNull();
    for (const bad of ["", "abc", "a@b", "a b@c.fr", "<x>@c.fr"]) expect(validateEmail(bad)).not.toBeNull();
  });
});

describe("validatePassword", () => {
  it("mirrors the server rules", () => {
    expect(validatePassword("short1!")).not.toBeNull();
    expect(validatePassword("onlyletterslong")).not.toBeNull();
    expect(validatePassword("12345678901")).not.toBeNull();
    expect(validatePassword("a1".repeat(37))).not.toBeNull();
    expect(validatePassword("Correct-Horse-9")).toBeNull();
  });
});

describe("validateFirstName", () => {
  it("accepts accents, hyphens and apostrophes only", () => {
    expect(validateFirstName("Léa-Rose")).toBeNull();
    expect(validateFirstName("D'Artagnan")).toBeNull();
    expect(validateFirstName("R2D2")).not.toBeNull();
    expect(validateFirstName("   ")).not.toBeNull();
    expect(validateFirstName("x".repeat(41))).not.toBeNull();
  });
});

describe("parseBirthdate", () => {
  it("builds an ISO date for an adult", () => {
    expect(parseBirthdate({ day: "5", month: "3", year: "1994" }, NOW)).toEqual({ iso: "1994-03-05", error: null });
  });
  it("enforces the 18+ rule to the day", () => {
    expect(parseBirthdate({ day: "15", month: "03", year: "2008" }, NOW).error).toBeNull();
    expect(parseBirthdate({ day: "16", month: "03", year: "2008" }, NOW).error).toMatch(/18 ans/);
  });
  it("rejects impossible or incomplete dates", () => {
    expect(parseBirthdate({ day: "31", month: "02", year: "1990" }, NOW).error).toMatch(/n'existe pas/);
    expect(parseBirthdate({ day: "", month: "02", year: "1990" }, NOW).error).not.toBeNull();
    expect(parseBirthdate({ day: "01", month: "01", year: "1800" }, NOW).error).not.toBeNull();
  });
});

describe("helpers", () => {
  it("computes age around birthdays", () => {
    expect(ageOn(new Date(Date.UTC(2000, 2, 15)), NOW)).toBe(26);
    expect(ageOn(new Date(Date.UTC(2000, 2, 16)), NOW)).toBe(25);
  });
  it("splits and sanitises", () => {
    expect(splitBirthdate("1994-03-05")).toEqual({ year: "1994", month: "03", day: "05" });
    expect(digitsOnly("1a2b3c4", 3)).toBe("123");
  });
});
