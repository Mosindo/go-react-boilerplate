// Run with: npm test (Node's built-in test runner, TypeScript stripped natively).
import assert from "node:assert/strict";
import { test } from "node:test";
import { distanceLabel, nameAge, parseBirthdate, relativeTime, validateEmail, validatePassword } from "../format.ts";

const now = new Date(2026, 5, 15, 12, 0, 0);

test("parseBirthdate accepts adults and formats ISO", () => {
  assert.deepEqual(parseBirthdate("5", "3", "1990", now), { iso: "1990-03-05" });
  assert.deepEqual(parseBirthdate("15", "06", "2008", now), { iso: "2008-06-15" });
});

test("parseBirthdate rejects minors, impossible dates and partial input", () => {
  assert.match(parseBirthdate("16", "06", "2008", now).error ?? "", /majeures/);
  assert.match(parseBirthdate("31", "02", "1990", now).error ?? "", /n'existe pas/);
  assert.match(parseBirthdate("1", "1", "90", now).error ?? "", /incomplète/);
  assert.ok(parseBirthdate("1", "1", "1900", now).error);
});

test("password and email validation mirror the server rules", () => {
  assert.equal(validatePassword("Password123"), null);
  assert.ok(validatePassword("short1"));
  assert.ok(validatePassword("onlyletters"));
  assert.equal(validateEmail(" a@b.co "), null);
  assert.ok(validateEmail("nope"));
});

test("labels never expose precise data", () => {
  assert.equal(distanceLabel(undefined), null);
  assert.equal(distanceLabel(1), "À moins de 2 km");
  assert.equal(distanceLabel(15), "À 15 km");
  assert.equal(nameAge("Léa"), "Léa");
  assert.equal(nameAge("Léa", 29), "Léa, 29");
});

test("relativeTime buckets", () => {
  const base = Date.UTC(2026, 0, 10, 12, 0, 0);
  assert.equal(relativeTime(new Date(base - 20_000).toISOString(), base), "À l'instant");
  assert.equal(relativeTime(new Date(base - 5 * 60_000).toISOString(), base), "5 min");
  assert.equal(relativeTime(new Date(base - 3 * 3_600_000).toISOString(), base), "3 h");
  assert.equal(relativeTime(new Date(base - 2 * 86_400_000).toISOString(), base), "2 j");
});
