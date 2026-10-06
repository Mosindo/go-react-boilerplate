import assert from "node:assert/strict";
import { test } from "node:test";
import { formatDistance, formatRelative, nameWithAge, previewText } from "./format.ts";

test("formatRelative", () => {
  const now = new Date(2026, 5, 15, 12, 0, 0).getTime();
  const ago = (ms: number) => new Date(now - ms).toISOString();
  assert.equal(formatRelative(ago(10_000), now), "à l'instant");
  assert.equal(formatRelative(ago(5 * 60_000), now), "5 min");
  assert.equal(formatRelative(ago(3 * 3_600_000), now), "3 h");
  assert.equal(formatRelative(ago(30 * 3_600_000), now), "hier");
  assert.match(formatRelative(ago(10 * 86_400_000), now), /^\d{2}\/\d{2}$/);
});

test("formatDistance shows only coarse values", () => {
  assert.equal(formatDistance(undefined), null);
  assert.equal(formatDistance(5), "À moins de 5 km");
  assert.equal(formatDistance(15), "À environ 15 km");
});

test("nameWithAge hides a hidden age", () => {
  assert.equal(nameWithAge("Léa", 28), "Léa, 28");
  assert.equal(nameWithAge("Léa", undefined), "Léa");
});

test("previewText flattens and truncates", () => {
  assert.equal(previewText("a\n\n b   c"), "a b c");
  assert.equal(previewText("x".repeat(100), 10), `${"x".repeat(9)}…`);
});
