import assert from "node:assert/strict";
import { test } from "node:test";
import { formatDayLabel, formatDistance, formatListTime, sameDay } from "./format.ts";

test("distance is always shown approximately", () => {
  assert.equal(formatDistance(undefined), null);
  assert.equal(formatDistance(1), "À moins de 2 km");
  assert.equal(formatDistance(25), "À 25 km");
});

test("list timestamps are relative to today", () => {
  const now = new Date(2026, 2, 10, 18, 0);
  assert.equal(formatListTime(new Date(2026, 2, 9, 12, 0).toISOString(), now), "Hier");
  assert.match(formatListTime(new Date(2026, 2, 10, 9, 5).toISOString(), now), /09:05/);
  assert.equal(formatDayLabel(new Date(2026, 2, 10, 1, 0).toISOString(), now), "Aujourd'hui");
  assert.equal(sameDay(new Date(2026, 2, 10, 1).toISOString(), new Date(2026, 2, 10, 23).toISOString()), true);
});
