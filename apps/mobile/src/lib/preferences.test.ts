import { describe, expect, it } from "vitest";
import { stepDistance, toggleGender } from "./preferences";

describe("stepDistance", () => {
  it("walks the ladder and stops at both ends", () => {
    expect(stepDistance(50, 1)).toBe(100);
    expect(stepDistance(50, -1)).toBe(25);
    expect(stepDistance(250, 1)).toBeNull();
    expect(stepDistance(null, 1)).toBeNull();
    expect(stepDistance(null, -1)).toBe(250);
    expect(stepDistance(5, -1)).toBe(5);
  });
});

describe("toggleGender", () => {
  it("never leaves the list empty", () => {
    expect(toggleGender(["man"], "man")).toEqual(["man"]);
    expect(toggleGender(["man", "woman"], "man")).toEqual(["woman"]);
    expect(toggleGender(["man"], "woman")).toEqual(["man", "woman"]);
  });
});
