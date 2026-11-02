import assert from "node:assert/strict";
import { test } from "node:test";
import {
  birthDateProblem,
  clamp,
  firstNameProblem,
  isValidEmail,
  isoToDisplayDate,
  normalizeEmail,
  parseBirthDate,
  passwordProblem
} from "../src/lib/validation.ts";

const NOW = new Date(Date.UTC(2026, 5, 15, 12));

test("normalizeEmail trims and lowercases", () => {
  assert.equal(normalizeEmail("  Jane@Example.COM "), "jane@example.com");
});

test("isValidEmail", () => {
  assert.ok(isValidEmail(" a@b.co "));
  for (const bad of ["", "a", "a@b", "a b@c.de", "@b.de", "a@.de"]) assert.equal(isValidEmail(bad), false, bad);
});

test("passwordProblem mirrors the API rules", () => {
  assert.equal(passwordProblem("Password1"), null);
  assert.ok(passwordProblem("short1"));
  assert.ok(passwordProblem("onlyletters"));
  assert.ok(passwordProblem("12345678"));
  assert.ok(passwordProblem("a1".repeat(40)));
});

test("parseBirthDate accepts both formats and rejects impossible dates", () => {
  assert.deepEqual(parseBirthDate("15/06/2000", NOW), { iso: "2000-06-15", age: 26 });
  assert.deepEqual(parseBirthDate("2000-06-16", NOW), { iso: "2000-06-16", age: 25 });
  assert.deepEqual(parseBirthDate("1/2/1990", NOW), { iso: "1990-02-01", age: 36 });
  for (const bad of ["31/02/2000", "00/01/2000", "15/13/2000", "abc", "", "2000/06/15x"]) {
    assert.equal(parseBirthDate(bad, NOW), null, bad);
  }
});

test("birthDateProblem enforces 18+ exactly on the birthday", () => {
  assert.equal(birthDateProblem("15/06/2008", NOW), null);
  assert.ok(birthDateProblem("16/06/2008", NOW));
  assert.ok(birthDateProblem("01/01/1900", NOW));
  assert.ok(birthDateProblem("nope", NOW));
});

test("isoToDisplayDate", () => {
  assert.equal(isoToDisplayDate("2000-06-05"), "05/06/2000");
  assert.equal(isoToDisplayDate("garbage"), "garbage");
});

test("firstNameProblem", () => {
  assert.equal(firstNameProblem("Zoé"), null);
  assert.ok(firstNameProblem("   "));
  assert.ok(firstNameProblem("a".repeat(41)));
});

test("clamp", () => {
  assert.equal(clamp(5, 1, 3), 3);
  assert.equal(clamp(-1, 1, 3), 1);
  assert.equal(clamp(2, 1, 3), 2);
});
