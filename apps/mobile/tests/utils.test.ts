import assert from "node:assert/strict";
import { test } from "node:test";
import {
  ageFromIso,
  formatDistance,
  formatRelative,
  isoToFrenchDate,
  maskDate,
  parseFrenchDate
} from "../src/utils/format.ts";
import { looksLikeEmail, passwordProblem } from "../src/utils/validation.ts";

test("parseFrenchDate accepts valid dates and rejects impossible ones", () => {
  assert.equal(parseFrenchDate("15/06/1995"), "1995-06-15");
  assert.equal(parseFrenchDate("29/02/2024"), "2024-02-29");
  assert.equal(parseFrenchDate("29/02/2023"), null);
  assert.equal(parseFrenchDate("31/04/2000"), null);
  assert.equal(parseFrenchDate("1995-06-15"), null);
  assert.equal(isoToFrenchDate("1995-06-15"), "15/06/1995");
});

test("maskDate inserts slashes while typing", () => {
  assert.equal(maskDate("1"), "1");
  assert.equal(maskDate("1506"), "15/06");
  assert.equal(maskDate("15061995"), "15/06/1995");
  assert.equal(maskDate("15/06/1995123"), "15/06/1995");
});

test("ageFromIso counts full years", () => {
  const now = new Date(2026, 5, 14); // 14 June 2026
  assert.equal(ageFromIso("2008-06-15", now), 17);
  assert.equal(ageFromIso("2008-06-14", now), 18);
});

test("formatDistance never reveals sub-kilometre precision", () => {
  assert.equal(formatDistance(null), null);
  assert.equal(formatDistance(1), "À moins de 2 km");
  assert.equal(formatDistance(12), "À 12 km");
});

test("formatRelative produces short labels", () => {
  const now = new Date(2026, 0, 10, 12, 0, 0);
  assert.equal(formatRelative(new Date(2026, 0, 10, 11, 59, 30).toISOString(), now), "à l'instant");
  assert.equal(formatRelative(new Date(2026, 0, 10, 11, 30).toISOString(), now), "il y a 30 min");
  assert.equal(formatRelative(new Date(2026, 0, 9, 18, 0).toISOString(), now), "hier");
  assert.equal(formatRelative(new Date(2026, 0, 2, 18, 0).toISOString(), now), "02/01");
});

test("password and e-mail validation mirror the server rules", () => {
  assert.equal(passwordProblem("short1"), "8 caractères minimum.");
  assert.equal(passwordProblem("onlyletters"), "Ajoute au moins une lettre et un chiffre.");
  assert.equal(passwordProblem("12345678"), "Ajoute au moins une lettre et un chiffre.");
  assert.equal(passwordProblem("motdepasse1"), null);
  assert.equal(passwordProblem("éèàçùâ12"), null);
  assert.equal(passwordProblem("a1".repeat(37)), "72 caractères maximum.");
  assert.ok(looksLikeEmail(" camille@example.fr "));
  assert.ok(!looksLikeEmail("camille@example"));
  assert.ok(!looksLikeEmail("cam ille@example.fr"));
});
