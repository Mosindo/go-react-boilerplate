import { ageOn, parseBirthDate, splitIsoDate, validateEmail, validateFirstName, validatePassword } from "../lib/validation";

const today = new Date(Date.UTC(2026, 5, 15));

describe("ageOn", () => {
  it("handles birthdays that have not happened yet this year", () => {
    expect(ageOn("2008-06-15", today)).toBe(18);
    expect(ageOn("2008-06-16", today)).toBe(17);
    expect(ageOn("1990-01-01", today)).toBe(36);
  });

  it("rejects impossible dates", () => {
    expect(ageOn("2001-02-30", today)).toBeNull();
    expect(ageOn("not-a-date", today)).toBeNull();
    expect(ageOn("2001-13-01", today)).toBeNull();
  });
});

describe("parseBirthDate", () => {
  it("accepts an adult and zero-pads the parts", () => {
    expect(parseBirthDate({ day: "5", month: "3", year: "1995" }, today)).toEqual({ iso: "1995-03-05", error: null });
  });

  it("refuses minors", () => {
    const result = parseBirthDate({ day: "16", month: "6", year: "2008" }, today);
    expect(result.iso).toBeNull();
    expect(result.error).toMatch(/18 ans/);
  });

  it("refuses malformed input", () => {
    expect(parseBirthDate({ day: "", month: "1", year: "1990" }, today).iso).toBeNull();
    expect(parseBirthDate({ day: "1", month: "1", year: "90" }, today).iso).toBeNull();
    expect(parseBirthDate({ day: "31", month: "2", year: "1990" }, today).error).toMatch(/n'existe pas/);
  });

  it("refuses implausible ages", () => {
    expect(parseBirthDate({ day: "1", month: "1", year: "1900" }, today).iso).toBeNull();
  });

  it("round-trips through splitIsoDate", () => {
    expect(splitIsoDate("1995-03-05")).toEqual({ day: "5", month: "3", year: "1995" });
  });
});

describe("field validators", () => {
  it("validates e-mail addresses", () => {
    expect(validateEmail("  a@b.co ")).toBeNull();
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
  });

  it("validates passwords", () => {
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("x".repeat(129))).not.toBeNull();
  });

  it("validates first names, accents included", () => {
    expect(validateFirstName("Zoë-Marie")).toBeNull();
    expect(validateFirstName("O'Neil")).toBeNull();
    expect(validateFirstName("  ")).not.toBeNull();
    expect(validateFirstName("R2D2")).not.toBeNull();
    expect(validateFirstName("<b>x</b>")).not.toBeNull();
  });
});
