import { describe, expect, it } from "vitest";
import type { Completion } from "../api/types";
import { firstIncompleteStep } from "./onboarding";

const base: Completion = { profile: false, preferences: false, location: false, photos: false, complete: false };

describe("firstIncompleteStep", () => {
  it("starts at the beginning for a new account", () => expect(firstIncompleteStep(base)).toBe(0));
  it("goes to photos once the profile exists", () => expect(firstIncompleteStep({ ...base, profile: true })).toBe(1));
  it("then to preferences", () => expect(firstIncompleteStep({ ...base, profile: true, photos: true })).toBe(3));
  it("then to location", () => expect(firstIncompleteStep({ ...base, profile: true, photos: true, preferences: true })).toBe(4));
  it("falls back to the bio step when only the server completion flags are satisfied", () => {
    expect(firstIncompleteStep({ profile: true, photos: true, preferences: true, location: true, complete: true })).toBe(2);
  });
});
