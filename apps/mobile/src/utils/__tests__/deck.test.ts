import type { PublicProfile } from "../../api/types";
import { DECK_REFILL_THRESHOLD, applyBatch, mergeCandidates, needsRefill, swipeDecision } from "../deck";

const profile = (id: string): PublicProfile => ({
  id,
  firstName: id,
  age: 30,
  gender: "woman",
  bio: "",
  city: "",
  distanceKm: null,
  interests: [],
  photos: []
});

describe("mergeCandidates", () => {
  it("appends only new, not-yet-swiped profiles", () => {
    const queue = [profile("a"), profile("b")];
    const merged = mergeCandidates(queue, [profile("b"), profile("c"), profile("d")], new Set(["d"]));
    expect(merged.map((p) => p.id)).toEqual(["a", "b", "c"]);
  });

  it("returns the same array when nothing is new (no needless re-render)", () => {
    const queue = [profile("a")];
    expect(mergeCandidates(queue, [profile("a")], new Set())).toBe(queue);
  });
});

describe("needsRefill", () => {
  it("refills when low, but not while fetching or after the server ran dry", () => {
    expect(needsRefill(DECK_REFILL_THRESHOLD, false, false)).toBe(true);
    expect(needsRefill(DECK_REFILL_THRESHOLD + 1, false, false)).toBe(false);
    expect(needsRefill(0, true, false)).toBe(false);
    expect(needsRefill(0, false, true)).toBe(false);
  });
});

describe("swipeDecision", () => {
  const width = 400;
  it("commits past the distance threshold", () => {
    expect(swipeDecision(150, 0, width)).toBe("like");
    expect(swipeDecision(-150, 0, width)).toBe("pass");
  });

  it("commits on a quick flick even over a short distance", () => {
    expect(swipeDecision(60, 1.2, width)).toBe("like");
    expect(swipeDecision(-60, -1.2, width)).toBe("pass");
  });

  it("springs back on a small, slow drag", () => {
    expect(swipeDecision(50, 0.2, width)).toBeNull();
    expect(swipeDecision(-50, -0.2, width)).toBeNull();
    expect(swipeDecision(10, 2, width)).toBeNull();
  });
});

describe("applyBatch", () => {
  it("adds fresh profiles and keeps refilling", () => {
    const result = applyBatch([profile("a")], [profile("b")], new Set());
    expect(result.queue.map((p) => p.id)).toEqual(["a", "b"]);
    expect(result.exhausted).toBe(false);
  });

  it("stops refilling when the batch adds nobody (prevents an endless refetch loop)", () => {
    const queue = [profile("a"), profile("b")];
    expect(applyBatch(queue, [profile("a"), profile("b")], new Set()).exhausted).toBe(true);
    expect(applyBatch(queue, [], new Set()).exhausted).toBe(true);
    expect(applyBatch(queue, [profile("c")], new Set(["c"])).exhausted).toBe(true);
  });
});
