import assert from "node:assert/strict";
import { test } from "node:test";
import { ageAt, firstNameProblem, isValidEmail, parseBirthdate, passwordProblem } from "./validation.ts";

const now = new Date(2026, 5, 15);

test("birthdate parsing enforces the 18+ rule", () => {
  assert.equal(parseBirthdate("15", "06", "2008", now).iso, "2008-06-15");
  assert.match(parseBirthdate("16", "06", "2008", now).error ?? "", /majeures/);
  assert.match(parseBirthdate("31", "02", "1990", now).error ?? "", /n'existe pas/);
  assert.match(parseBirthdate("1", "1", "90", now).error ?? "", /incomplète/);
});

test("age is computed in completed years", () => {
  assert.equal(ageAt(new Date(1990, 5, 16), now), 35);
  assert.equal(ageAt(new Date(1990, 5, 15), now), 36);
});

test("credentials validation mirrors the API policy", () => {
  assert.equal(isValidEmail("  Ana@Lueur.app "), true);
  assert.equal(isValidEmail("ana@localhost"), false);
  assert.equal(passwordProblem("Password123"), null);
  assert.notEqual(passwordProblem("onlyletters"), null);
  assert.notEqual(passwordProblem("short1"), null);
});

test("first names allow letters, hyphens and apostrophes only", () => {
  assert.equal(firstNameProblem("Anne-Sophie"), null);
  assert.equal(firstNameProblem("D'Arcy"), null);
  assert.notEqual(firstNameProblem("R2D2"), null);
  assert.notEqual(firstNameProblem("   "), null);
});
