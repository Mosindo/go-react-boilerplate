import type { PublicProfile } from "../../api";
import { deckReducer, initialDeckState, MIN_QUEUE_SIZE, needsRefill } from "./deck";
import { resolveRelease, resolveTap } from "./swipeController";

const profile = (userId: string): PublicProfile => ({
  userId,
  firstName: userId,
  age: 30,
  bio: "",
  city: "",
  distanceKm: null,
  interests: [],
  photos: [],
  sharedInterests: []
});

describe("deckReducer", () => {
  it("appends new profiles and dedupes by userId", () => {
    const first = deckReducer(initialDeckState, { type: "received", profiles: [profile("a"), profile("b")] });
    const second = deckReducer(first, { type: "received", profiles: [profile("b"), profile("c")] });
    expect(second.queue.map((p) => p.userId)).toEqual(["a", "b", "c"]);
    expect(second.exhausted).toBe(false);
  });

  it("never re-adds a profile that was already swiped", () => {
    let state = deckReducer(initialDeckState, { type: "received", profiles: [profile("a")] });
    state = deckReducer(state, { type: "swiped", userId: "a" });
    state = deckReducer(state, { type: "received", profiles: [profile("a")] });
    expect(state.queue).toEqual([]);
    expect(state.exhausted).toBe(true);
  });

  it("marks the deck exhausted when the server has nothing new, and resume clears it", () => {
    const empty = deckReducer(initialDeckState, { type: "received", profiles: [] });
    expect(empty.exhausted).toBe(true);
    expect(needsRefill(empty)).toBe(false);
    expect(needsRefill(deckReducer(empty, { type: "resume" }))).toBe(true);
  });

  it("asks for a refill when fewer than the minimum cards remain", () => {
    const ids = Array.from({ length: MIN_QUEUE_SIZE }, (_unused, i) => `u${i}`);
    let state = deckReducer(initialDeckState, { type: "received", profiles: ids.map(profile) });
    expect(needsRefill(state)).toBe(false);
    state = deckReducer(state, { type: "swiped", userId: "u0" });
    expect(needsRefill(state)).toBe(true);
  });

  it("restores a card on top after a failed swipe, once", () => {
    let state = deckReducer(initialDeckState, { type: "received", profiles: [profile("a"), profile("b")] });
    state = deckReducer(state, { type: "swiped", userId: "a" });
    state = deckReducer(state, { type: "restore", profile: profile("a") });
    state = deckReducer(state, { type: "restore", profile: profile("a") });
    expect(state.queue.map((p) => p.userId)).toEqual(["a", "b"]);
  });
});

describe("swipe gestures", () => {
  it("resolves releases", () => {
    expect(resolveRelease({ dx: 150, dy: 0, vx: 0 }, 360)).toBe("like");
    expect(resolveRelease({ dx: -150, dy: 0, vx: 0 }, 360)).toBe("pass");
    expect(resolveRelease({ dx: 20, dy: 0, vx: 1.2 }, 360)).toBe("like");
    expect(resolveRelease({ dx: -20, dy: 0, vx: -1.2 }, 360)).toBe("pass");
    expect(resolveRelease({ dx: 40, dy: 10, vx: 0.1 }, 360)).toBe("cancel");
    expect(resolveRelease({ dx: 2, dy: 3, vx: 0 }, 360)).toBe("tap");
  });

  it("resolves taps on edges and centre", () => {
    expect(resolveTap(10, 360, 3)).toBe("previous");
    expect(resolveTap(350, 360, 3)).toBe("next");
    expect(resolveTap(180, 360, 3)).toBe("open");
    expect(resolveTap(10, 360, 1)).toBe("open");
  });
});
