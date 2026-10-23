import {
  decideSwipe,
  isTap,
  remainingDeck,
  rotationDegrees,
  shouldPrefetch,
  stampOpacity,
  swipeThreshold
} from "../src/lib/swipe";
import { computeResize, isTooSmall } from "../src/lib/photoResize";

describe("swipe decision", () => {
  const W = 400;

  it("scales the threshold with the container but keeps a minimum", () => {
    expect(swipeThreshold(400)).toBeCloseTo(112);
    expect(swipeThreshold(200)).toBe(90);
  });

  it("likes right, passes left once past the threshold", () => {
    expect(decideSwipe({ dx: 150, dy: 0, vx: 0 }, W)).toBe("like");
    expect(decideSwipe({ dx: -150, dy: 10, vx: 0 }, W)).toBe("pass");
  });

  it("cancels short slow drags", () => {
    expect(decideSwipe({ dx: 60, dy: 0, vx: 0.1 }, W)).toBeNull();
    expect(decideSwipe({ dx: -60, dy: 0, vx: -0.1 }, W)).toBeNull();
  });

  it("accepts fast flicks over a short distance", () => {
    expect(decideSwipe({ dx: 60, dy: 0, vx: 1.2 }, W)).toBe("like");
    expect(decideSwipe({ dx: -60, dy: 0, vx: -1.2 }, W)).toBe("pass");
    expect(decideSwipe({ dx: 20, dy: 0, vx: 1.2 }, W)).toBeNull();
  });

  it("ignores mostly vertical drags", () => {
    expect(decideSwipe({ dx: 80, dy: 300, vx: 0 }, W)).toBeNull();
  });

  it("detects taps", () => {
    expect(isTap({ dx: 2, dy: -3, vx: 0 })).toBe(true);
    expect(isTap({ dx: 20, dy: 0, vx: 0 })).toBe(false);
  });

  it("computes stamp opacity and rotation", () => {
    expect(stampOpacity(0, "like", W)).toBe(0);
    expect(stampOpacity(swipeThreshold(W) / 2, "like", W)).toBeCloseTo(0.5);
    expect(stampOpacity(10_000, "like", W)).toBe(1);
    expect(stampOpacity(-swipeThreshold(W), "pass", W)).toBe(1);
    expect(stampOpacity(50, "pass", W)).toBe(0);
    expect(rotationDegrees(W, W)).toBe(12);
    expect(rotationDegrees(-10 * W, W)).toBe(-12);
  });
});

describe("deck", () => {
  const cards = [{ userId: "a" }, { userId: "b" }, { userId: "c" }];
  it("hides locally swiped cards", () => {
    expect(remainingDeck(cards, new Set(["a", "c"]))).toEqual([{ userId: "b" }]);
    expect(remainingDeck(cards, new Set())).toHaveLength(3);
  });
  it("prefetches when the deck runs low", () => {
    expect(shouldPrefetch(3)).toBe(true);
    expect(shouldPrefetch(4)).toBe(false);
  });
});

describe("photo resize", () => {
  it("resizes only the longest side above 1080", () => {
    expect(computeResize(4000, 3000)).toEqual({ width: 1080 });
    expect(computeResize(3000, 4000)).toEqual({ height: 1080 });
    expect(computeResize(1080, 800)).toBeNull();
  });
  it("rejects images under 64px", () => {
    expect(isTooSmall(63, 500)).toBe(true);
    expect(isTooSmall(64, 64)).toBe(false);
  });
});
