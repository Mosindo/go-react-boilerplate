import { ageOn, checkBirthdate, isValidEmail, isValidFirstName, passwordProblem, roundCoordinate } from "../utils/validation";

const now = new Date(2026, 5, 15);

describe("checkBirthdate", () => {
  it("accepts adults and returns an ISO date", () => {
    const result = checkBirthdate("10", "3", "1995", now);
    expect(result).toEqual({ ok: true, iso: "1995-03-10", age: 31 });
  });

  it("rejects minors, including the day before their 18th birthday", () => {
    expect(checkBirthdate("16", "6", "2008", now).ok).toBe(false);
    expect(checkBirthdate("15", "6", "2008", now).ok).toBe(true);
  });

  it("rejects impossible or partial dates", () => {
    expect(checkBirthdate("30", "2", "1990", now).ok).toBe(false);
    expect(checkBirthdate("1", "1", "90", now).ok).toBe(false);
    expect(checkBirthdate("", "", "", now).ok).toBe(false);
    expect(checkBirthdate("1", "1", "2030", now).ok).toBe(false);
  });
});

describe("field validators", () => {
  it("validates emails", () => {
    expect(isValidEmail(" lea@mail.fr ")).toBe(true);
    expect(isValidEmail("lea@mail")).toBe(false);
  });

  it("enforces the password policy", () => {
    expect(passwordProblem("short1")).not.toBeNull();
    expect(passwordProblem("onlyletters")).not.toBeNull();
    expect(passwordProblem("Lettres123")).toBeNull();
  });

  it("validates first names", () => {
    expect(isValidFirstName("Marie-Ève")).toBe(true);
    expect(isValidFirstName("<b>")).toBe(false);
    expect(isValidFirstName("")).toBe(false);
  });

  it("computes ages and rounds coordinates to ~1 km", () => {
    expect(ageOn(new Date(2000, 5, 16), now)).toBe(25);
    expect(roundCoordinate(48.856613)).toBe(48.86);
  });
});
