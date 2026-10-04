import { ageFromIso, isoToBirthInput, maskBirthInput, parseBirthDate } from "../dates";

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe("maskBirthInput", () => {
  it("inserts slashes and strips non digits", () => {
    expect(maskBirthInput("1")).toBe("1");
    expect(maskBirthInput("170")).toBe("17/0");
    expect(maskBirthInput("17051994")).toBe("17/05/1994");
    expect(maskBirthInput("17/05/1994999")).toBe("17/05/1994");
    expect(maskBirthInput("ab12")).toBe("12");
  });
});

describe("ageFromIso", () => {
  it("counts whole years and handles birthdays not yet reached", () => {
    expect(ageFromIso("2008-06-15", NOW)).toBe(18);
    expect(ageFromIso("2008-06-16", NOW)).toBe(17);
    expect(ageFromIso("1990-01-01", NOW)).toBe(36);
  });
});

describe("parseBirthDate", () => {
  it("accepts adults and returns an ISO date", () => {
    expect(parseBirthDate("17/05/1994", NOW)).toEqual({ iso: "1994-05-17", age: 32 });
  });
  it("accepts someone turning 18 today, rejects tomorrow", () => {
    expect(parseBirthDate("15/06/2008", NOW)).toMatchObject({ age: 18 });
    expect(parseBirthDate("16/06/2008", NOW)).toHaveProperty("error");
  });
  it("rejects impossible calendar dates", () => {
    expect(parseBirthDate("31/02/1990", NOW)).toEqual({ error: "That date doesn't exist." });
    expect(parseBirthDate("29/02/2001", NOW)).toHaveProperty("error");
    expect(parseBirthDate("29/02/2000", NOW)).toHaveProperty("iso");
  });
  it("rejects bad formats, future dates and implausible ages", () => {
    expect(parseBirthDate("1990-01-01", NOW)).toHaveProperty("error");
    expect(parseBirthDate("1/1/1990", NOW)).toHaveProperty("error");
    expect(parseBirthDate("01/01/2030", NOW)).toEqual({ error: "Birth date can't be in the future." });
    expect(parseBirthDate("01/01/1900", NOW)).toHaveProperty("error");
  });
  it("explains the adult-only rule", () => {
    const result = parseBirthDate("01/01/2015", NOW);
    expect("error" in result && result.error).toMatch(/18/);
  });
});

describe("isoToBirthInput", () => {
  it("round-trips", () => {
    expect(isoToBirthInput("1994-05-17")).toBe("17/05/1994");
    expect(isoToBirthInput("garbage")).toBe("");
  });
});
