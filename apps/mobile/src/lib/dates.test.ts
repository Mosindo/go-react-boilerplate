import { ageOn, formatBirthDateInput, formatListDate, isoToDisplay, parseBirthDate } from "./dates";

describe("parseBirthDate", () => {
  it("converts a valid French date to ISO", () => {
    expect(parseBirthDate("05/03/1994")).toBe("1994-03-05");
  });
  it("rejects impossible or malformed dates", () => {
    expect(parseBirthDate("31/02/1994")).toBeNull();
    expect(parseBirthDate("1994-03-05")).toBeNull();
    expect(parseBirthDate("5/3/1994")).toBeNull();
    expect(parseBirthDate("")).toBeNull();
  });
});

describe("formatBirthDateInput", () => {
  it("adds slashes progressively and strips junk", () => {
    expect(formatBirthDateInput("3")).toBe("3");
    expect(formatBirthDateInput("311")).toBe("31/1");
    expect(formatBirthDateInput("31121990")).toBe("31/12/1990");
    expect(formatBirthDateInput("31/12/1990999")).toBe("31/12/1990");
    expect(formatBirthDateInput("ab12")).toBe("12");
  });
});

describe("ageOn", () => {
  it("counts completed years only", () => {
    const now = new Date(2026, 5, 15);
    expect(ageOn("2008-06-16", now)).toBe(17);
    expect(ageOn("2008-06-15", now)).toBe(18);
  });
});

describe("isoToDisplay", () => {
  it("reverses the date", () => {
    expect(isoToDisplay("1994-03-05")).toBe("05/03/1994");
    expect(isoToDisplay("bad")).toBe("");
  });
});

describe("formatListDate", () => {
  const now = new Date(2026, 5, 15, 12, 0);
  it("shows the time today, Hier yesterday, dd/mm before", () => {
    expect(formatListDate(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListDate(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListDate(new Date(2026, 5, 1, 8, 0).toISOString(), now)).toBe("01/06");
  });
});
