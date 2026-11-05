import { describe, expect, it } from "vitest";
import { formatTimestamp, isoToDisplay, parseBirthDate } from "./dates";

const now = new Date(Date.UTC(2026, 5, 15, 12));

describe("parseBirthDate", () => {
  it("parses a valid date and computes the age", () => {
    expect(parseBirthDate("15/06/2008", now)).toEqual({ iso: "2008-06-15", age: 18 });
    expect(parseBirthDate("16/06/2008", now)).toEqual({ iso: "2008-06-16", age: 17 });
  });
  it("rejects malformed and impossible dates", () => {
    for (const bad of ["", "2008-06-15", "31/02/2000", "00/01/2000", "1/1/2000", "15/13/2000"]) {
      expect(parseBirthDate(bad, now)).toBeNull();
    }
  });
  it("accepts leap days", () => {
    expect(parseBirthDate("29/02/2000", now)?.iso).toBe("2000-02-29");
  });
});

describe("isoToDisplay", () => {
  it("round-trips", () => {
    expect(isoToDisplay("1995-03-07")).toBe("07/03/1995");
    expect(isoToDisplay("nope")).toBe("");
  });
});

describe("formatTimestamp", () => {
  it("shows time for today and 'hier' for yesterday", () => {
    const today = new Date(2026, 5, 15, 9, 5);
    const local = new Date(2026, 5, 15, 18, 0);
    expect(formatTimestamp(today.toISOString(), local)).toMatch(/09[:h]05/);
    expect(formatTimestamp(new Date(2026, 5, 14, 23, 0).toISOString(), local)).toBe("hier");
  });
  it("returns an empty string for invalid input", () => {
    expect(formatTimestamp("garbage")).toBe("");
  });
});
