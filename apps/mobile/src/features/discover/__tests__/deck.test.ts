import type { PublicProfile } from "../../../api/types";
import { deckReducer, initialDeck, shouldPrefetch } from "../deck";

const p = (userId: string): PublicProfile => ({
  userId,
  displayName: userId,
  age: 30,
  gender: "woman",
  bio: "",
  city: "",
  interests: [],
  photos: [],
  distanceKm: null,
  commonInterests: 0
});

describe("deckReducer", () => {
  it("appends new profiles, ignores ones already seen and flags exhaustion", () => {
    let s = deckReducer(initialDeck, { type: "append", profiles: [p("a"), p("b")] });
    expect(s.cards.map((c) => c.userId)).toEqual(["a", "b"]);
    s = deckReducer(s, { type: "swipe", userId: "a" });
    s = deckReducer(s, { type: "append", profiles: [p("a"), p("c")] });
    expect(s.cards.map((c) => c.userId)).toEqual(["b", "c"]);
    expect(s.exhausted).toBe(false);
    s = deckReducer(s, { type: "append", profiles: [p("a")] });
    expect(s.exhausted).toBe(true);
  });

  it("rolls a failed swipe back to the top of the deck", () => {
    let s = deckReducer(initialDeck, { type: "append", profiles: [p("a"), p("b")] });
    s = deckReducer(s, { type: "swipe", userId: "a" });
    s = deckReducer(s, { type: "rollback", profile: p("a") });
    expect(s.cards.map((c) => c.userId)).toEqual(["a", "b"]);
    expect(deckReducer(s, { type: "rollback", profile: p("a") })).toBe(s);
  });

  it("removes blocked users and resets", () => {
    const s = deckReducer(initialDeck, { type: "append", profiles: [p("a"), p("b")] });
    expect(deckReducer(s, { type: "remove", userId: "b" }).cards).toHaveLength(1);
    expect(deckReducer(s, { type: "reset" })).toBe(initialDeck);
  });
});

describe("shouldPrefetch", () => {
  it("fetches when 3 or fewer cards remain, unless loading or exhausted", () => {
    expect(shouldPrefetch(3, false, false)).toBe(true);
    expect(shouldPrefetch(4, false, false)).toBe(false);
    expect(shouldPrefetch(0, true, false)).toBe(false);
    expect(shouldPrefetch(0, false, true)).toBe(false);
  });
});
