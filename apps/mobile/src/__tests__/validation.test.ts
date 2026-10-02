import {
  ageOn,
  birthDateError,
  isoToBirthInput,
  isValidEmail,
  maskBirthInput,
  parseBirthDate,
  passwordError,
} from "../lib/validation";

describe("birth date", () => {
  const now = new Date("2026-06-15T12:00:00");

  it("parses real dates and rejects impossible ones", () => {
    expect(parseBirthDate("15/06/1990")).toBe("1990-06-15");
    expect(parseBirthDate("31/02/1990")).toBeNull();
    expect(parseBirthDate("1990-06-15")).toBeNull();
    expect(parseBirthDate("")).toBeNull();
  });

  it("refuses minors, accepts the 18th birthday", () => {
    expect(birthDateError("16/06/2008", now)).toMatch(/majeures/);
    expect(birthDateError("15/06/2008", now)).toBeNull();
    expect(birthDateError("01/01/1900", now)).not.toBeNull();
  });

  it("computes age around birthdays", () => {
    expect(ageOn(new Date("2000-06-16T00:00:00"), now)).toBe(25);
    expect(ageOn(new Date("2000-06-15T00:00:00"), now)).toBe(26);
  });

  it("masks typing and round-trips ISO dates", () => {
    expect(maskBirthInput("15061990")).toBe("15/06/1990");
    expect(maskBirthInput("1506")).toBe("15/06");
    expect(maskBirthInput("15/06/19901234")).toBe("15/06/1990");
    expect(isoToBirthInput("1990-06-15")).toBe("15/06/1990");
  });
});

describe("credentials", () => {
  it("validates e-mail shape", () => {
    expect(isValidEmail(" a@b.co ")).toBe(true);
    expect(isValidEmail("a@b")).toBe(false);
    expect(isValidEmail("nope")).toBe(false);
  });

  it("enforces the bcrypt-safe password window", () => {
    expect(passwordError("short")).not.toBeNull();
    expect(passwordError("long-enough")).toBeNull();
    expect(passwordError("é".repeat(40))).not.toBeNull(); // 80 bytes
  });
});
