import assert from "node:assert/strict";
import { test } from "node:test";
import {
  clampAgeRange,
  formatBirthdateInput,
  isValidEmail,
  isoToBirthdateInput,
  parseBirthdate,
  passwordProblem,
  validFirstName
} from "./validation.ts";

const NOW = new Date(2026, 5, 15); // 15 June 2026, local time

test("isValidEmail", () => {
  assert.equal(isValidEmail("  a@b.co "), true);
  for (const bad of ["", "plain", "a@b", "a b@c.co", "a@b.", "Name <a@b.co>"]) {
    assert.equal(isValidEmail(bad), false, bad);
  }
});

test("passwordProblem enforces server limits", () => {
  assert.equal(passwordProblem("short"), "Au moins 8 caractères.");
  assert.equal(passwordProblem("a".repeat(73)), "72 caractères maximum.");
  assert.equal(passwordProblem("é".repeat(40)), "72 caractères maximum."); // 80 bytes
  assert.equal(passwordProblem("long enough"), null);
});

test("formatBirthdateInput inserts separators and ignores junk", () => {
  assert.equal(formatBirthdateInput("1"), "1");
  assert.equal(formatBirthdateInput("1403"), "14/03");
  assert.equal(formatBirthdateInput("14031994"), "14/03/1994");
  assert.equal(formatBirthdateInput("14/03/1994999"), "14/03/1994");
  assert.equal(formatBirthdateInput("ab"), "");
});

test("parseBirthdate accepts adults only and real dates only", () => {
  assert.deepEqual(parseBirthdate("15/06/2008", NOW), { ok: true, iso: "2008-06-15", age: 18 });
  const tomorrow = parseBirthdate("16/06/2008", NOW);
  assert.equal(tomorrow.ok, false);
  assert.equal(parseBirthdate("31/02/1990", NOW).ok, false);
  assert.equal(parseBirthdate("1990-02-01", NOW).ok, false);
  assert.equal(parseBirthdate("01/01/1900", NOW).ok, false);
  assert.equal(parseBirthdate("01/01/2030", NOW).ok, false);
  const ok = parseBirthdate("14/03/1994", NOW);
  assert.equal(ok.ok && ok.age, 32);
});

test("isoToBirthdateInput round-trips", () => {
  assert.equal(isoToBirthdateInput("1994-03-14"), "14/03/1994");
  assert.equal(isoToBirthdateInput("nonsense"), "");
});

test("validFirstName", () => {
  assert.equal(validFirstName("Anne-Laure"), null);
  assert.equal(validFirstName("  Léa "), null);
  assert.notEqual(validFirstName(""), null);
  assert.notEqual(validFirstName("R2D2"), null);
  assert.notEqual(validFirstName("<b>"), null);
  assert.notEqual(validFirstName("x".repeat(41)), null);
});

test("clampAgeRange keeps a valid, ordered range", () => {
  assert.deepEqual(clampAgeRange(10, 120), { min: 18, max: 99 });
  assert.deepEqual(clampAgeRange(40, 25), { min: 25, max: 40 });
});
