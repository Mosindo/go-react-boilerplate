import { describe, expect, it } from "vitest";
import { firstMissingStep, ONBOARDING_STEPS, progressFraction } from "./onboarding";

describe("firstMissingStep", () => {
  it("resumes at the first blocking step", () => {
    expect(firstMissingStep({ hasProfile: false, hasLocation: false, photoCount: 0 })).toBe(0);
    expect(firstMissingStep({ hasProfile: true, hasLocation: false, photoCount: 0 })).toBe(3);
    expect(firstMissingStep({ hasProfile: true, hasLocation: true, photoCount: 0 })).toBe(4);
    expect(firstMissingStep({ hasProfile: true, hasLocation: false, photoCount: 2 })).toBe(3);
    expect(firstMissingStep({ hasProfile: true, hasLocation: true, photoCount: 2 })).toBe(
      ONBOARDING_STEPS.length - 1
    );
  });
});

describe("progressFraction", () => {
  it("is bounded", () => {
    expect(progressFraction(0)).toBeCloseTo(0.2);
    expect(progressFraction(4)).toBe(1);
    expect(progressFraction(99)).toBe(1);
    expect(progressFraction(-3)).toBeCloseTo(0.2);
  });
});
