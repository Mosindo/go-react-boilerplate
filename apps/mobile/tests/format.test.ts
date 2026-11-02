import assert from "node:assert/strict";
import { test } from "node:test";
import { formatActivity, formatClock, formatDistance, pluralize } from "../src/lib/format.ts";

test("formatDistance never shows more precision than the API gives", () => {
  assert.equal(formatDistance(10), "À moins de 10 km");
  assert.equal(formatDistance(null), null);
  assert.equal(formatDistance(undefined), null);
});

test("formatActivity: clock today, Hier, weekday, then date", () => {
  const now = new Date(2026, 5, 15, 18, 0); // Monday
  assert.equal(formatActivity(new Date(2026, 5, 15, 9, 5).toISOString(), now), "09:05");
  assert.equal(formatActivity(new Date(2026, 5, 14, 23, 0).toISOString(), now), "Hier");
  assert.equal(formatActivity(new Date(2026, 5, 11, 12, 0).toISOString(), now), "jeu.");
  assert.equal(formatActivity(new Date(2026, 4, 1, 12, 0).toISOString(), now), "01/05");
  assert.equal(formatActivity("invalid", now), "");
});

test("formatClock", () => {
  assert.equal(formatClock(new Date(2026, 0, 1, 7, 3).toISOString()), "07:03");
});

test("pluralize", () => {
  assert.equal(pluralize(1, "match", "matchs"), "match");
  assert.equal(pluralize(2, "match", "matchs"), "matchs");
});
