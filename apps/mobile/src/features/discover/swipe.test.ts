import {
  appendBatch,
  decideSwipe,
  exitTarget,
  nextPhotoIndex,
  rotationFor,
  shouldPrefetch,
  stampOpacity,
  tapStep
} from "./swipe";
import { formatDistance } from "./distance";

const W = 400;

describe("decideSwipe", () => {
  it("commits like/pass beyond the distance threshold", () => {
    expect(decideSwipe({ dx: 130, dy: 0, vx: 0 }, W)).toBe("like");
    expect(decideSwipe({ dx: -130, dy: 10, vx: 0 }, W)).toBe("pass");
  });
  it("ignores small slow drags", () => {
    expect(decideSwipe({ dx: 60, dy: 0, vx: 0.2 }, W)).toBe("none");
    expect(decideSwipe({ dx: 0, dy: 0, vx: 0 }, W)).toBe("none");
  });
  it("commits fast flings with a minimum distance", () => {
    expect(decideSwipe({ dx: 50, dy: 0, vx: 1.2 }, W)).toBe("like");
    expect(decideSwipe({ dx: -50, dy: 0, vx: -1.2 }, W)).toBe("pass");
    expect(decideSwipe({ dx: 10, dy: 0, vx: 2 }, W)).toBe("none");
  });
  it("does not commit mostly vertical drags", () => {
    expect(decideSwipe({ dx: 100, dy: 300, vx: 0 }, W)).toBe("none");
  });
  it("does not fling right when dragged left", () => {
    expect(decideSwipe({ dx: -50, dy: 0, vx: 1.2 }, W)).toBe("none");
  });
});

describe("gesture visuals", () => {
  it("stamps scale with drag and clamp", () => {
    expect(stampOpacity(0, W)).toEqual({ like: 0, pass: 0 });
    expect(stampOpacity(60, W).like).toBeCloseTo(0.5);
    expect(stampOpacity(-1000, W)).toEqual({ like: 0, pass: 1 });
  });
  it("rotates proportionally and is bounded", () => {
    expect(rotationFor(0, W)).toBe(0);
    expect(rotationFor(W * 5, W)).toBeCloseTo(24);
    expect(rotationFor(-W * 5, W)).toBeCloseTo(-24);
  });
  it("exits off-screen in the swipe direction", () => {
    expect(exitTarget("like", W)).toBeGreaterThan(W);
    expect(exitTarget("pass", W)).toBeLessThan(-W);
  });
});

describe("deck helpers", () => {
  it("prefetches only when fewer than 3 cards remain and idle", () => {
    expect(shouldPrefetch(2, true, false)).toBe(true);
    expect(shouldPrefetch(3, true, false)).toBe(false);
    expect(shouldPrefetch(0, false, false)).toBe(false);
    expect(shouldPrefetch(1, true, true)).toBe(false);
  });
  it("appends batches without duplicates or already swiped users", () => {
    const deck = [{ userId: "a" }, { userId: "b" }];
    const next = appendBatch(
      deck,
      [{ userId: "b" }, { userId: "c" }, { userId: "d" }, { userId: "c" }],
      new Set(["d"])
    );
    expect(next.map((p) => p.userId)).toEqual(["a", "b", "c"]);
    expect(appendBatch(deck, [{ userId: "a" }], new Set())).toBe(deck);
  });
  it("moves the photo index within bounds", () => {
    expect(nextPhotoIndex(0, 3, -1)).toBe(0);
    expect(nextPhotoIndex(0, 3, 1)).toBe(1);
    expect(nextPhotoIndex(2, 3, 1)).toBe(2);
    expect(nextPhotoIndex(0, 0, 1)).toBe(0);
  });
  it("maps tap position to a step", () => {
    expect(tapStep(50, 400)).toBe(-1);
    expect(tapStep(300, 400)).toBe(1);
  });
});

describe("formatDistance", () => {
  it("formats rounded kilometres", () => {
    expect(formatDistance(5)).toBe("~5 km away");
    expect(formatDistance(4.6)).toBe("~5 km away");
    expect(formatDistance(0.4)).toBe("Less than 1 km away");
  });
  it("returns null when unknown", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(undefined)).toBeNull();
    expect(formatDistance(Number.NaN)).toBeNull();
  });
});
