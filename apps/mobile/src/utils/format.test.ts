import assert from "node:assert/strict";
import { test } from "node:test";
import { ageFromBirthdate, formatDistance, formatListTime, isValidEmail, passwordProblem } from "./format.ts";

const now = new Date(2026, 5, 15, 12, 0, 0); // 15 June 2026

test("ageFromBirthdate enforces the 18+ rule precisely", () => {
  assert.equal(ageFromBirthdate("2008-06-15", now), 18);
  assert.equal(ageFromBirthdate("2008-06-16", now), 17);
  assert.equal(ageFromBirthdate("1990-01-01", now), 36);
});

test("ageFromBirthdate rejects impossible dates", () => {
  assert.equal(ageFromBirthdate("2001-02-30", now), null);
  assert.equal(ageFromBirthdate("15/06/2000", now), null);
  assert.equal(ageFromBirthdate("", now), null);
});

test("distance is always approximate and never zero", () => {
  assert.equal(formatDistance(null), null);
  assert.equal(formatDistance(1), "à moins de 1 km");
  assert.equal(formatDistance(12), "à 12 km");
});

test("passwordProblem mirrors the server policy", () => {
  assert.equal(passwordProblem("Password123"), null);
  assert.notEqual(passwordProblem("short1"), null);
  assert.notEqual(passwordProblem("onlyletters"), null);
  assert.notEqual(passwordProblem("12345678"), null);
});

test("isValidEmail", () => {
  assert.ok(isValidEmail(" someone@example.org "));
  assert.ok(!isValidEmail("someone@"));
});

test("formatListTime uses relative labels", () => {
  assert.equal(formatListTime(new Date(2026, 5, 14, 9, 0).toISOString(), now), "hier");
  assert.match(formatListTime(new Date(2026, 5, 15, 9, 5).toISOString(), now), /09:05/);
});
