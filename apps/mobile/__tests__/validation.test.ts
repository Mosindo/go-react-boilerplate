import { t } from "../src/i18n/fr";
import {
  calculateAge,
  formatDayLabel,
  formatListTime,
  formatMessageTime,
  isAdult,
  isDifferentDay,
  parseBirthDate,
  splitBirthDate
} from "../src/lib/dates";
import { formatDistance, messagePreview, moveItem, toggleInList } from "../src/lib/format";
import { draftToInput, emptyDraft, validateDraft } from "../src/lib/profileDraft";
import {
  normalizeEmail,
  validateBio,
  validateBirthDate,
  validateCity,
  validateEmail,
  validateFirstName,
  validateMessage,
  validatePassword,
  validatePasswordConfirmation
} from "../src/lib/validation";

const NOW = new Date(2025, 5, 15, 12, 0, 0); // 15 June 2025

describe("email", () => {
  it("normalizes and validates", () => {
    expect(normalizeEmail("  Alice@Example.COM ")).toBe("alice@example.com");
    expect(validateEmail("  alice@example.com ")).toBeNull();
    expect(validateEmail("")).toBe(t.validation.required);
    expect(validateEmail("alice@")).toBe(t.validation.emailInvalid);
    expect(validateEmail("a b@c.de")).toBe(t.validation.emailInvalid);
  });
});

describe("password", () => {
  it("enforces 8-72 chars with a letter and a digit", () => {
    expect(validatePassword("abc12345")).toBeNull();
    expect(validatePassword("short1")).toBe(t.validation.passwordLength);
    expect(validatePassword("a1".repeat(40))).toBe(t.validation.passwordLength);
    expect(validatePassword("abcdefgh")).toBe(t.validation.passwordComposition);
    expect(validatePassword("12345678")).toBe(t.validation.passwordComposition);
    expect(validatePassword("")).toBe(t.validation.required);
  });

  it("checks confirmation", () => {
    expect(validatePasswordConfirmation("a", "a")).toBeNull();
    expect(validatePasswordConfirmation("a", "b")).toBe(t.validation.passwordMismatch);
  });
});

describe("text fields", () => {
  it("validates lengths", () => {
    expect(validateFirstName("  ")).toBe(t.validation.required);
    expect(validateFirstName("x".repeat(41))).toBe(t.validation.nameTooLong);
    expect(validateFirstName("Léa")).toBeNull();
    expect(validateCity("x".repeat(81))).toBe(t.validation.cityTooLong);
    expect(validateBio("x".repeat(501))).toBe(t.validation.bioTooLong);
    expect(validateBio("")).toBeNull();
    expect(validateMessage("   ")).toBe(t.validation.required);
    expect(validateMessage("x".repeat(2001))).toBe(t.validation.messageTooLong);
    expect(validateMessage(" hello ")).toBeNull();
  });
});

describe("age and birth date", () => {
  it("calculates full years", () => {
    expect(calculateAge("2000-06-15", NOW)).toBe(25);
    expect(calculateAge("2000-06-16", NOW)).toBe(24);
    expect(calculateAge("2000-12-31", NOW)).toBe(24);
  });

  it("is adult exactly on the 18th birthday", () => {
    expect(isAdult("2007-06-15", NOW)).toBe(true);
    expect(isAdult("2007-06-16", NOW)).toBe(false);
  });

  it("parses real calendar dates only", () => {
    expect(parseBirthDate("5", "3", "1995", NOW)).toBe("1995-03-05");
    expect(parseBirthDate("31", "02", "1995", NOW)).toBeNull();
    expect(parseBirthDate("29", "02", "1996", NOW)).toBe("1996-02-29");
    expect(parseBirthDate("29", "02", "1997", NOW)).toBeNull();
    expect(parseBirthDate("1", "13", "1995", NOW)).toBeNull();
    expect(parseBirthDate("1", "1", "95", NOW)).toBeNull();
    expect(parseBirthDate("1", "1", "2030", NOW)).toBeNull();
    expect(parseBirthDate("a", "1", "1990", NOW)).toBeNull();
  });

  it("rejects minors and implausible years", () => {
    expect(validateBirthDate("15", "06", "2007", NOW)).toBeNull();
    expect(validateBirthDate("16", "06", "2007", NOW)).toBe(t.validation.birthUnderage);
    expect(validateBirthDate("01", "01", "1900", NOW)).toBe(t.validation.birthTooOld);
    expect(validateBirthDate("", "", "", NOW)).toBe(t.validation.birthInvalid);
  });

  it("splits ISO dates for editing", () => {
    expect(splitBirthDate("1995-03-05")).toEqual({ day: "05", month: "03", year: "1995" });
  });
});

describe("date formatting", () => {
  const at = (y: number, m: number, d: number, h = 9, min = 5) =>
    new Date(y, m, d, h, min).toISOString();

  it("formats message clock time", () => {
    expect(formatMessageTime(at(2025, 5, 15, 9, 5))).toBe("09:05");
    expect(formatMessageTime("nope")).toBe("");
  });

  it("formats list time relative to now", () => {
    expect(formatListTime(at(2025, 5, 15, 8, 30), NOW)).toBe("08:30");
    expect(formatListTime(at(2025, 5, 14, 23, 59), NOW)).toBe(t.time.yesterday);
    expect(formatListTime(at(2025, 5, 11), NOW)).toBe("mer.");
    expect(formatListTime(at(2025, 2, 3), NOW)).toBe("3 mars");
    expect(formatListTime(at(2024, 11, 24), NOW)).toBe("24 déc. 2024");
    expect(formatListTime("", NOW)).toBe("");
  });

  it("labels chat days", () => {
    expect(formatDayLabel(at(2025, 5, 15), NOW)).toBe(t.chat.today);
    expect(formatDayLabel(at(2025, 5, 14), NOW)).toBe(t.chat.yesterday);
    expect(formatDayLabel(at(2025, 0, 2), NOW)).toBe("2 janv.");
    expect(isDifferentDay(at(2025, 5, 15, 1), at(2025, 5, 15, 23))).toBe(false);
    expect(isDifferentDay(at(2025, 5, 15), at(2025, 5, 14))).toBe(true);
  });
});

describe("format helpers", () => {
  it("formats distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(10)).toBe("À environ 10 km");
    expect(formatDistance(0)).toBe("À environ 1 km");
  });

  it("builds conversation previews", () => {
    expect(messagePreview(null, "me")).toBe(t.matches.newMatch);
    expect(messagePreview({ content: "yo", senderId: "me" }, "me")).toBe("Vous : yo");
    expect(messagePreview({ content: "yo", senderId: "x" }, "me")).toBe("yo");
  });

  it("toggles and reorders lists", () => {
    expect(toggleInList(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleInList(["a", "b"], "a")).toEqual(["b"]);
    expect(toggleInList(["a", "b"], "c", 2)).toEqual(["a", "b"]);
    expect(moveItem([1, 2, 3], 2, 0)).toEqual([3, 1, 2]);
    expect(moveItem([1, 2, 3], 0, -5)).toEqual([1, 2, 3]);
    expect(moveItem([1, 2, 3], 9, 0)).toEqual([1, 2, 3]);
  });
});

describe("profile draft", () => {
  const valid = {
    ...emptyDraft,
    firstName: " Léa ",
    day: "5",
    month: "3",
    year: "1995",
    gender: "woman" as const,
    city: " Lyon ",
    bio: " Salut ",
    interests: ["a", "b"]
  };

  it("builds a trimmed ProfileInput", () => {
    expect(draftToInput(valid, NOW)).toEqual({
      firstName: "Léa",
      birthDate: "1995-03-05",
      gender: "woman",
      bio: "Salut",
      city: "Lyon",
      latitude: null,
      longitude: null,
      interests: ["a", "b"]
    });
  });

  it("returns null and errors when invalid", () => {
    expect(draftToInput({ ...valid, year: "2015" }, NOW)).toBeNull();
    const errors = validateDraft({ ...emptyDraft }, NOW);
    expect(errors.firstName).toBeTruthy();
    expect(errors.gender).toBeTruthy();
    expect(errors.city).toBeTruthy();
  });
});
