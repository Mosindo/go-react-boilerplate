import assert from "node:assert/strict";
import { test } from "node:test";
import { ageFromBirthDate, distanceLabel, formatBirthDateInput, validateBirthDate } from "./format";

const today = new Date(Date.UTC(2026, 2, 10));

test("ageFromBirthDate handles birthdays and invalid dates", () => {
  assert.equal(ageFromBirthDate("2008-03-10", today), 18);
  assert.equal(ageFromBirthDate("2008-03-11", today), 17);
  assert.equal(ageFromBirthDate("2001-02-30", today), null);
  assert.equal(ageFromBirthDate("abc", today), null);
});

test("validateBirthDate enforces adult age", () => {
  assert.equal(validateBirthDate("1990-01-01"), null);
  assert.match(validateBirthDate("1990-13-01") ?? "", /valide/);
  const recent = `${new Date().getUTCFullYear() - 10}-01-01`;
  assert.match(validateBirthDate(recent) ?? "", /18 ans/);
});

test("formatBirthDateInput builds a dashed date progressively", () => {
  assert.equal(formatBirthDateInput("1990"), "1990");
  assert.equal(formatBirthDateInput("199005"), "1990-05");
  assert.equal(formatBirthDateInput("19900517xx"), "1990-05-17");
});

test("distanceLabel only exposes buckets", () => {
  assert.equal(distanceLabel(null), null);
  assert.equal(distanceLabel(10), "À moins de 10 km");
});
