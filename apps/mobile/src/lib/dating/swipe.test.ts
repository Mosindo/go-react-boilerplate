import { describe, expect, it } from "vitest";
import {
  classifySwipeFailure,
  decideSwipe,
  flyOffTarget,
  likeStampOpacity,
  passStampOpacity
} from "./swipe";

describe("decideSwipe", () => {
  it("likes past the right threshold and passes past the left threshold", () => {
    expect(decideSwipe(120, 0, 400)).toBe("like");
    expect(decideSwipe(-120, 0, 400)).toBe("pass");
  });
  it("springs back under the threshold", () => {
    expect(decideSwipe(60, 0.1, 400)).toBeNull();
    expect(decideSwipe(-60, -0.1, 400)).toBeNull();
  });
  it("accepts a fast flick with enough travel", () => {
    expect(decideSwipe(50, 0.9, 400)).toBe("like");
    expect(decideSwipe(-50, -0.9, 400)).toBe("pass");
  });
  it("ignores a fast jitter with almost no travel", () => {
    expect(decideSwipe(5, 2, 400)).toBeNull();
  });
});

describe("stamps", () => {
  it("follows the drag distance and clamps", () => {
    expect(likeStampOpacity(0, 400)).toBe(0);
    expect(likeStampOpacity(50, 400)).toBeCloseTo(0.5);
    expect(likeStampOpacity(500, 400)).toBe(1);
    expect(likeStampOpacity(-50, 400)).toBe(0);
    expect(passStampOpacity(-50, 400)).toBeCloseTo(0.5);
    expect(passStampOpacity(50, 400)).toBe(0);
  });
});

describe("flyOffTarget", () => {
  it("goes off screen on the matching side", () => {
    expect(flyOffTarget("like", 400)).toBeGreaterThan(400);
    expect(flyOffTarget("pass", 400)).toBeLessThan(-400);
  });
});

describe("classifySwipeFailure", () => {
  it("maps statuses", () => {
    expect(classifySwipeFailure(409)).toBe("already-swiped");
    expect(classifySwipeFailure(404)).toBe("unavailable");
    expect(classifySwipeFailure(422)).toBe("profile-incomplete");
    expect(classifySwipeFailure(500)).toBe("retry");
    expect(classifySwipeFailure(null)).toBe("retry");
  });
});
