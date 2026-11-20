import assert from "node:assert/strict";
import { test } from "node:test";
import { ageOn, isValidEmail, parseBirthdate, passwordProblem, toggle } from "../src/lib/validation.ts";

const today = new Date(2026, 5, 15);

test("email validation trims and lowercases", () => {
  assert.equal(isValidEmail("  Alice@Mail.Test "), true);
  assert.equal(isValidEmail("alice@"), false);
  assert.equal(isValidEmail("alice mail.test"), false);
});

test("password policy matches the API", () => {
  assert.equal(passwordProblem("Password1"), null);
  assert.notEqual(passwordProblem("short1"), null);
  assert.notEqual(passwordProblem("onlyletters"), null);
  assert.notEqual(passwordProblem("12345678"), null);
});

test("birthdate parsing enforces adulthood and real dates", () => {
  const ok = parseBirthdate("15", "6", "2008", today);
  assert.equal(ok.ok, true);
  if (ok.ok) {
    assert.equal(ok.iso, "2008-06-15");
    assert.equal(ok.age, 18);
  }
  assert.equal(parseBirthdate("16", "06", "2008", today).ok, false);
  assert.equal(parseBirthdate("30", "02", "1990", today).ok, false);
  assert.equal(parseBirthdate("1", "1", "90", today).ok, false);
});

test("age computation handles birthdays", () => {
  assert.equal(ageOn(new Date(2000, 1, 29), new Date(2026, 1, 28)), 25);
  assert.equal(ageOn(new Date(2000, 1, 29), new Date(2026, 2, 1)), 26);
});

test("toggle respects the maximum", () => {
  assert.deepEqual(toggle([1, 2], 3, 2), [1, 2]);
  assert.deepEqual(toggle([1, 2], 2), [1]);
  assert.deepEqual(toggle([1], 2), [1, 2]);
});
