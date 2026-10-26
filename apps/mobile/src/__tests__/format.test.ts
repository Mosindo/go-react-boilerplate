import {
  ageFromIso,
  distanceLabel,
  maskDate,
  parseBirthDate,
  timeLabel,
  toDisplayDate,
  validEmail,
  validPassword
} from "../lib/format";

describe("birth date helpers", () => {
  it("masks typing into JJ/MM/AAAA", () => {
    expect(maskDate("1")).toBe("1");
    expect(maskDate("1203")).toBe("12/03");
    expect(maskDate("12031990")).toBe("12/03/1990");
    expect(maskDate("12/03/1990999")).toBe("12/03/1990");
    expect(maskDate("ab12")).toBe("12");
  });

  it("parses valid dates and rejects impossible ones", () => {
    expect(parseBirthDate("14/07/1994")).toBe("1994-07-14");
    expect(parseBirthDate("29/02/2000")).toBe("2000-02-29");
    expect(parseBirthDate("29/02/2001")).toBeNull();
    expect(parseBirthDate("31/04/1990")).toBeNull();
    expect(parseBirthDate("1/1/1990")).toBeNull();
    expect(parseBirthDate("")).toBeNull();
  });

  it("converts back for display and computes age on the birthday boundary", () => {
    expect(toDisplayDate("1994-07-14")).toBe("14/07/1994");
    const now = new Date(2026, 5, 15); // 15 June 2026
    expect(ageFromIso("2008-06-15", now)).toBe(18);
    expect(ageFromIso("2008-06-16", now)).toBe(17);
  });
});

describe("validation", () => {
  it("validates emails", () => {
    expect(validEmail(" a@b.co ")).toBe(true);
    expect(validEmail("nope")).toBe(false);
    expect(validEmail("a@b")).toBe(false);
  });

  it("applies the same password rules as the API", () => {
    expect(validPassword("Password1")).toBeNull();
    expect(validPassword("short1")).not.toBeNull();
    expect(validPassword("lettersonly")).not.toBeNull();
    expect(validPassword("12345678")).not.toBeNull();
    expect(validPassword("a1".repeat(40))).not.toBeNull();
  });
});

describe("labels", () => {
  it("shows distance as an approximation", () => {
    expect(distanceLabel(null)).toBeNull();
    expect(distanceLabel(5)).toBe("À moins de 5 km");
    expect(distanceLabel(25)).toBe("À moins de 25 km");
  });

  it("formats message times", () => {
    const now = new Date(2026, 5, 15, 18, 0);
    expect(timeLabel(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toMatch(/09:05|9:05/);
    expect(timeLabel("not a date", now)).toBe("");
  });
});
