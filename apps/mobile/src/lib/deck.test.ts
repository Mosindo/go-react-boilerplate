import { describe, expect, it } from "vitest";
import { DECK_REFILL_THRESHOLD, excludeIds, mergeCards, shouldRefill } from "./deck";

const card = (id: string) => ({ userId: id });

describe("mergeCards", () => {
  it("appends only new, undecided cards and keeps order", () => {
    const queue = [card("a"), card("b")];
    const merged = mergeCards(queue, [card("b"), card("c"), card("d"), card("c")], new Set(["d"]));
    expect(merged.map((c) => c.userId)).toEqual(["a", "b", "c"]);
  });

  it("returns the same array when nothing is new", () => {
    const queue = [card("a")];
    expect(mergeCards(queue, [card("a")], new Set())).toBe(queue);
  });
});

describe("shouldRefill", () => {
  it("refills when the queue is low and not already loading or exhausted", () => {
    expect(shouldRefill(DECK_REFILL_THRESHOLD, false, false)).toBe(true);
    expect(shouldRefill(DECK_REFILL_THRESHOLD + 1, false, false)).toBe(false);
    expect(shouldRefill(0, true, false)).toBe(false);
    expect(shouldRefill(0, false, true)).toBe(false);
  });
});

describe("excludeIds", () => {
  it("never sends more than the API accepts", () => {
    const queue = Array.from({ length: 80 }, (_, i) => card(`u${i}`));
    expect(excludeIds(queue)).toHaveLength(50);
  });
});
