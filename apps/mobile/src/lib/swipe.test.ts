import { describe, expect, it } from "vitest";
import { decideSwipe } from "./swipe";

describe("decideSwipe", () => {
  it("likes on a long right drag and passes on a long left drag", () => {
    expect(decideSwipe(150, 0, 400)).toBe("like");
    expect(decideSwipe(-150, 0, 400)).toBe("pass");
  });
  it("accepts a quick flick", () => {
    expect(decideSwipe(60, 1.2, 400)).toBe("like");
    expect(decideSwipe(-60, -1.2, 400)).toBe("pass");
  });
  it("snaps back on a small, slow drag", () => {
    expect(decideSwipe(30, 0.1, 400)).toBeNull();
    expect(decideSwipe(-80, -0.2, 400)).toBeNull();
  });
});
