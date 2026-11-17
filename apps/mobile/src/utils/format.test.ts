import { test } from "node:test";
import assert from "node:assert/strict";
import {
  ageFromIso,
  dayLabel,
  distanceLabel,
  isValidEmail,
  isoToFrenchDate,
  maskFrenchDate,
  parseFrenchDate,
  passwordProblem,
  shortTime
} from "./format.ts";

test("parseFrenchDate validates real calendar dates", () => {
  assert.equal(parseFrenchDate("29/02/2000"), "2000-02-29");
  assert.equal(parseFrenchDate("29/02/2001"), null);
  assert.equal(parseFrenchDate("31/04/1990"), null);
  assert.equal(parseFrenchDate("1990-04-01"), null);
  assert.equal(isoToFrenchDate("1990-04-01"), "01/04/1990");
});

test("maskFrenchDate inserts separators while typing", () => {
  assert.equal(maskFrenchDate("12"), "12");
  assert.equal(maskFrenchDate("1203"), "12/03");
  assert.equal(maskFrenchDate("12031990999"), "12/03/1990");
  assert.equal(maskFrenchDate("12/0a3"), "12/03");
});

test("ageFromIso counts full years", () => {
  const now = new Date(2026, 10, 17);
  assert.equal(ageFromIso("2008-11-17", now), 18);
  assert.equal(ageFromIso("2008-11-18", now), 17);
});

test("distanceLabel never reveals sub-kilometre precision", () => {
  assert.equal(distanceLabel(1), "À moins de 1 km");
  assert.equal(distanceLabel(12), "À 12 km");
  assert.equal(distanceLabel(null), null);
});

test("time labels", () => {
  const now = new Date(2026, 10, 17, 18, 0);
  assert.equal(shortTime(new Date(2026, 10, 17, 9, 5).toISOString(), now), "09:05");
  assert.equal(shortTime(new Date(2026, 10, 16, 9, 5).toISOString(), now), "Hier");
  assert.equal(shortTime(new Date(2026, 9, 1).toISOString(), now), "01/10");
  assert.equal(dayLabel(new Date(2026, 10, 17, 1).toISOString(), now), "Aujourd'hui");
});

test("client-side validation mirrors the API", () => {
  assert.equal(isValidEmail(" a@b.co "), true);
  assert.equal(isValidEmail("a@b"), false);
  assert.equal(passwordProblem("short1"), "8 caractères minimum.");
  assert.equal(passwordProblem("onlyletters"), "Au moins une lettre et un chiffre.");
  assert.equal(passwordProblem("Password123"), null);
});
