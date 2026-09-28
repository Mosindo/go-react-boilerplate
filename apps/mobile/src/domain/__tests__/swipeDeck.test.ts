import type { Candidate } from "../../api/types";
import { deckReducer, initialDeck, shouldPrefetch, type DeckState } from "../swipeDeck";

function candidate(id: string): Candidate {
  return {
    userId: id,
    firstName: id,
    age: 30,
    bio: "",
    distanceKm: 5,
    locationLabel: "Paris",
    interests: [],
    sharedInterestCount: 0,
    photos: []
  };
}

function loaded(...ids: string[]): DeckState {
  return deckReducer(initialDeck, { type: "fetchSucceeded", items: ids.map(candidate) });
}

describe("swipe deck state machine", () => {
  it("appends new candidates and ignores duplicates", () => {
    const state = deckReducer(loaded("a", "b"), { type: "fetchSucceeded", items: [candidate("b"), candidate("c")] });
    expect(state.cards.map((card) => card.userId)).toEqual(["a", "b", "c"]);
    expect(state.exhausted).toBe(false);
  });

  it("marks the deck exhausted when the server has nothing new", () => {
    const state = deckReducer(loaded("a"), { type: "fetchSucceeded", items: [] });
    expect(state.exhausted).toBe(true);
  });

  it("removes the top card optimistically and tracks it in flight", () => {
    const state = deckReducer(loaded("a", "b"), { type: "swiped", userId: "a" });
    expect(state.cards.map((card) => card.userId)).toEqual(["b"]);
    expect(Object.keys(state.inFlight)).toEqual(["a"]);
  });

  it("ignores a swipe for a card that is not on top", () => {
    const start = loaded("a", "b");
    expect(deckReducer(start, { type: "swiped", userId: "b" })).toBe(start);
  });

  it("rolls a failed swipe back to the top of the deck", () => {
    const swiped = deckReducer(loaded("a", "b"), { type: "swiped", userId: "a" });
    const state = deckReducer(swiped, { type: "swipeRolledBack", userId: "a" });
    expect(state.cards.map((card) => card.userId)).toEqual(["a", "b"]);
    expect(state.inFlight).toEqual({});
    expect(state.seen).toEqual({});
  });

  it("keeps swiped profiles out of later batches, even while the request is pending", () => {
    const swiped = deckReducer(loaded("a"), { type: "swiped", userId: "a" });
    const pending = deckReducer(swiped, { type: "fetchSucceeded", items: [candidate("a"), candidate("z")] });
    expect(pending.cards.map((card) => card.userId)).toEqual(["z"]);
    const confirmed = deckReducer(pending, { type: "swipeConfirmed", userId: "a" });
    const later = deckReducer(confirmed, { type: "fetchSucceeded", items: [candidate("a")] });
    expect(later.cards.map((card) => card.userId)).toEqual(["z"]);
  });

  it("drops an unavailable profile without bringing it back", () => {
    const swiped = deckReducer(loaded("a"), { type: "swiped", userId: "a" });
    const state = deckReducer(swiped, { type: "swipeDropped", userId: "a" });
    expect(state.cards).toEqual([]);
    expect(state.inFlight).toEqual({});
    expect(state.seen).toEqual({ a: true });
  });

  it("prefetches when fewer than 3 cards remain and nothing blocks it", () => {
    expect(shouldPrefetch(loaded("a", "b"))).toBe(true);
    expect(shouldPrefetch(loaded("a", "b", "c"))).toBe(false);
    expect(shouldPrefetch(deckReducer(loaded("a"), { type: "fetchStarted" }))).toBe(false);
    expect(shouldPrefetch(deckReducer(loaded("a"), { type: "fetchFailed" }))).toBe(false);
    expect(shouldPrefetch(deckReducer(loaded("a"), { type: "fetchSucceeded", items: [] }))).toBe(false);
  });

  it("reset clears cards and flags but remembers swiped profiles", () => {
    const swiped = deckReducer(loaded("a", "b"), { type: "swiped", userId: "a" });
    const state = deckReducer(swiped, { type: "reset" });
    expect(state.cards).toEqual([]);
    expect(state.exhausted).toBe(false);
    expect(state.seen).toEqual({ a: true });
    expect(shouldPrefetch(state)).toBe(true);
  });
});
