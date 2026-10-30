import { describe, expect, it } from "vitest";
import { ageFromIso, formatBirthDateInput, parseBirthDate } from "./dates";
import { clamp, distanceLabel, timeLabel } from "./format";
import { validateBirthDate, validateEmail, validateNewPassword } from "./validation";

describe("dates", () => {
  it("parses real dates and rejects impossible ones", () => {
    expect(parseBirthDate("05/03/1992")).toBe("1992-03-05");
    expect(parseBirthDate("31/02/1992")).toBeNull();
    expect(parseBirthDate("1992-03-05")).toBeNull();
    expect(parseBirthDate("")).toBeNull();
  });

  it("computes age around the birthday", () => {
    const now = new Date(Date.UTC(2026, 5, 15));
    expect(ageFromIso("2008-06-16", now)).toBe(17);
    expect(ageFromIso("2008-06-15", now)).toBe(18);
  });

  it("formats typed digits progressively", () => {
    expect(formatBirthDateInput("1")).toBe("1");
    expect(formatBirthDateInput("120319")).toBe("12/03/19");
    expect(formatBirthDateInput("12031990999")).toBe("12/03/1990");
    expect(formatBirthDateInput("ab12")).toBe("12");
  });
});

describe("validation", () => {
  it("accepts adults only", () => {
    const now = new Date(Date.UTC(2026, 5, 15));
    expect(validateBirthDate("15/06/2008", now).iso).toBe("2008-06-15");
    expect(validateBirthDate("16/06/2008", now).error).toMatch(/18 ans/);
    expect(validateBirthDate("01/01/1900", now).error).not.toBeNull();
    expect(validateBirthDate("nope", now).error).not.toBeNull();
  });

  it("checks email and password rules like the server", () => {
    expect(validateEmail(" a@b.co ")).toBeNull();
    expect(validateEmail("a@b")).not.toBeNull();
    expect(validateEmail("")).not.toBeNull();
    expect(validateNewPassword("short")).not.toBeNull();
    expect(validateNewPassword("long-enough-1")).toBeNull();
    expect(validateNewPassword("é".repeat(40))).not.toBeNull();
  });
});

describe("format", () => {
  it("rounds distances the way the API does", () => {
    expect(distanceLabel(undefined)).toBeNull();
    expect(distanceLabel(1)).toBe("À moins de 1 km");
    expect(distanceLabel(15)).toBe("À environ 15 km");
  });

  it("labels timestamps", () => {
    const now = new Date("2026-06-15T12:00:00");
    expect(timeLabel("2026-06-15T09:05:00", now)).toMatch(/09:05|9:05/);
    expect(timeLabel("2026-05-01T09:05:00", now)).toMatch(/01\/05/);
  });

  it("clamps", () => {
    expect(clamp(5, 1, 3)).toBe(3);
    expect(clamp(-1, 1, 3)).toBe(1);
  });
});
