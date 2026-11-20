import assert from "node:assert/strict";
import { test } from "node:test";
import { errorMessage, formatDayLabel, formatDistance, formatListTime, formatRelative } from "../src/lib/format.ts";

const now = new Date(2026, 5, 15, 18, 30);

test("distance is always approximate", () => {
  assert.equal(formatDistance(null), null);
  assert.equal(formatDistance(1), "À moins de 1 km");
  assert.equal(formatDistance(25), "À 25 km");
});

test("list timestamps", () => {
  assert.equal(formatListTime(new Date(2026, 5, 15, 9, 5).toISOString(), now), "09:05");
  assert.equal(formatListTime(new Date(2026, 5, 14, 9, 5).toISOString(), now), "Hier");
  assert.equal(formatListTime(new Date(2026, 4, 2, 9, 5).toISOString(), now), "02/05");
  assert.equal(formatDayLabel(new Date(2026, 5, 15, 1).toISOString(), now), "Aujourd’hui");
  assert.equal(formatRelative(new Date(2026, 5, 15, 18, 10).toISOString(), now), "Il y a 20 min");
});

test("API error codes map to French messages", () => {
  assert.equal(errorMessage({ code: "invalid_credentials" }), "E-mail ou mot de passe incorrect.");
  assert.equal(errorMessage({ code: "unknown_code" }, "fallback"), "fallback");
  assert.equal(errorMessage(new Error("Erreur 500"), "fallback"), "fallback");
});
