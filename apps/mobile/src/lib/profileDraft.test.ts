import { describe, expect, it } from "vitest";
import { draftToInput, emptyDraft, toggleInterest, validateDraft } from "./profileDraft";

describe("validateDraft", () => {
  it("requires a first name and a gender", () => {
    const errors = validateDraft(emptyDraft);
    expect(errors.firstName).toBeDefined();
    expect(errors.gender).toBeDefined();
  });
  it("accepts a valid draft and rejects out-of-range heights", () => {
    const ok = { ...emptyDraft, firstName: "Léa", gender: "woman" as const };
    expect(validateDraft(ok)).toEqual({});
    expect(validateDraft({ ...ok, height: "90" }).height).toBeDefined();
    expect(validateDraft({ ...ok, height: "172" }).height).toBeUndefined();
    expect(validateDraft({ ...ok, height: "17x" }).height).toBeDefined();
  });
  it("counts characters, not bytes, for the bio", () => {
    const ok = { ...emptyDraft, firstName: "Léa", gender: "woman" as const };
    expect(validateDraft({ ...ok, bio: "é".repeat(500) }).bio).toBeUndefined();
    expect(validateDraft({ ...ok, bio: "é".repeat(501) }).bio).toBeDefined();
  });
});

describe("draftToInput", () => {
  it("trims, maps empty height to null and drops coordinates when clearing the location", () => {
    const input = draftToInput({
      ...emptyDraft,
      firstName: "  Léa ",
      gender: "woman",
      latitude: 48.85,
      longitude: 2.35,
      clearLocation: true
    });
    expect(input.firstName).toBe("Léa");
    expect(input.heightCm).toBeNull();
    expect(input.latitude).toBeUndefined();
    expect(input.clearLocation).toBe(true);
  });
});

describe("toggleInterest", () => {
  it("toggles and caps at 10", () => {
    expect(toggleInterest(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleInterest(["a", "b"], "a")).toEqual(["b"]);
    const full = Array.from({ length: 10 }, (_, i) => `i${i}`);
    expect(toggleInterest(full, "extra")).toEqual(full);
  });
});
