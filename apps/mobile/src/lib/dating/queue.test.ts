import { describe, expect, it } from "vitest";
import type { PublicProfile } from "../../api/models";
import {
  discoverQueueReducer,
  initialDiscoverQueue,
  shouldPrefetch,
  type DiscoverQueueState
} from "./queue";

function profile(id: string): PublicProfile {
  return {
    userId: id,
    firstName: id,
    age: 30,
    gender: "woman",
    bio: "",
    city: "",
    distanceKm: null,
    interests: [],
    photos: []
  };
}

describe("discoverQueueReducer", () => {
  it("appends and dedupes", () => {
    let s = discoverQueueReducer(initialDiscoverQueue, {
      type: "append",
      profiles: [profile("a"), profile("b"), profile("a")]
    });
    expect(s.queue.map((p) => p.userId)).toEqual(["a", "b"]);
    s = discoverQueueReducer(s, { type: "append", profiles: [profile("b"), profile("c")] });
    expect(s.queue.map((p) => p.userId)).toEqual(["a", "b", "c"]);
  });

  it("never re-shows a card already acted on", () => {
    let s = discoverQueueReducer(initialDiscoverQueue, { type: "append", profiles: [profile("a")] });
    s = discoverQueueReducer(s, { type: "act", userId: "a" });
    expect(s.queue).toEqual([]);
    s = discoverQueueReducer(s, { type: "append", profiles: [profile("a"), profile("b")] });
    expect(s.queue.map((p) => p.userId)).toEqual(["b"]);
  });

  it("restores a failed card on top and allows it again", () => {
    const start: DiscoverQueueState = discoverQueueReducer(initialDiscoverQueue, {
      type: "append",
      profiles: [profile("a"), profile("b")]
    });
    let s = discoverQueueReducer(start, { type: "act", userId: "a" });
    s = discoverQueueReducer(s, { type: "restore", profile: profile("a") });
    expect(s.queue.map((p) => p.userId)).toEqual(["a", "b"]);
    expect(s.seen).not.toContain("a");
  });

  it("prefetches at three or fewer cards", () => {
    expect(shouldPrefetch(4)).toBe(false);
    expect(shouldPrefetch(3)).toBe(true);
    expect(shouldPrefetch(0)).toBe(true);
  });
});
