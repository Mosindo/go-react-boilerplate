import assert from "node:assert/strict";
import { test } from "node:test";
import { emptyDraft, hasErrors, toggle, validateAbout, validateBasics, validatePreferences } from "./draft";

test("basics require name, adult birth date and gender", () => {
  assert.equal(hasErrors(validateBasics(emptyDraft())), true);
  const ok = { ...emptyDraft(), firstName: "Alex", birthDate: "1992-04-12", gender: "woman" as const };
  assert.equal(hasErrors(validateBasics(ok)), false);
  assert.ok(validateBasics({ ...ok, birthDate: "2999-01-01" }).birthDate);
});

test("about enforces length limits", () => {
  assert.ok(validateAbout({ ...emptyDraft(), bio: "x".repeat(501) }).bio);
  assert.equal(hasErrors(validateAbout(emptyDraft())), false);
});

test("preferences need a target and a coherent age range", () => {
  assert.ok(validatePreferences(emptyDraft()).interestedIn);
  const base = { ...emptyDraft(), interestedIn: ["man" as const] };
  assert.equal(hasErrors(validatePreferences(base)), false);
  assert.ok(validatePreferences({ ...base, ageMin: 40, ageMax: 30 }).ageMin);
});

test("toggle adds and removes", () => {
  assert.deepEqual(toggle(["a"], "b"), ["a", "b"]);
  assert.deepEqual(toggle(["a", "b"], "a"), ["b"]);
});
