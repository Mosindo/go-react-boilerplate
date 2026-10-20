import type { Profile } from "../api/types";
import { formatDistance } from "../lib/format";
import { dedupeById, flattenPages } from "../lib/pagination";
import { appendProfiles } from "../lib/swipeQueue";
import { photoUri, toWebSocketUrl } from "../lib/url";
import { nextPhotoIndex } from "../features/discover/ProfileCard";
import { resolveSwipe } from "../features/discover/SwipeDeck";
import { DISTANCE_STEPS, nextDistance, toggleValue } from "../features/profile/forms";
import { backoffDelay, parseRealtimeEvent } from "../realtime/RealtimeProvider";

const profile = (id: string): Profile => ({
  userId: id,
  firstName: id,
  age: 30,
  gender: "woman",
  bio: "",
  city: "",
  interests: [],
  photos: [],
  distanceKm: null
});

describe("urls", () => {
  it("derives websocket urls", () => {
    expect(toWebSocketUrl("http://192.168.1.2:18080/")).toBe("ws://192.168.1.2:18080/ws");
    expect(toWebSocketUrl("https://api.example.org")).toBe("wss://api.example.org/ws");
  });
  it("joins photo urls", () => {
    expect(photoUri("http://x:1/", "/photos/a/file")).toBe("http://x:1/photos/a/file");
  });
});

describe("pagination and queue", () => {
  it("dedupes and flattens pages", () => {
    expect(dedupeById([{ id: "a" }, { id: "a" }, { id: "b" }])).toHaveLength(2);
    expect(
      flattenPages([
        { items: [{ id: "a" }], nextCursor: "c" },
        { items: [{ id: "a" }, { id: "b" }], nextCursor: null }
      ])
    ).toEqual([{ id: "a" }, { id: "b" }]);
    expect(flattenPages(undefined)).toEqual([]);
  });
  it("never re-queues handled or known profiles", () => {
    const queue = [profile("a")];
    const result = appendProfiles(queue, [profile("a"), profile("b"), profile("c")], new Set(["c"]));
    expect(result.map((p) => p.userId)).toEqual(["a", "b"]);
    expect(appendProfiles(queue, [profile("a")], new Set())).toBe(queue);
  });
});

describe("misc helpers", () => {
  it("formats distances coarsely", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toMatch(/Less than 1/);
    expect(formatDistance(25)).toBe("25 km away");
  });
  it("resolves swipe direction", () => {
    expect(resolveSwipe(200, 100)).toBe("like");
    expect(resolveSwipe(-200, 100)).toBe("pass");
    expect(resolveSwipe(50, 100)).toBeNull();
  });
  it("bounds photo navigation", () => {
    expect(nextPhotoIndex(0, 3, -1)).toBe(0);
    expect(nextPhotoIndex(2, 3, 1)).toBe(2);
    expect(nextPhotoIndex(1, 3, 1)).toBe(2);
    expect(nextPhotoIndex(0, 1, 1)).toBe(0);
  });
  it("steps distance within bounds", () => {
    expect(nextDistance(DISTANCE_STEPS[0], -1)).toBe(DISTANCE_STEPS[0]);
    expect(nextDistance(25, 1)).toBe(50);
    expect(nextDistance(999, 1)).toBe(200);
  });
  it("toggles list values", () => {
    expect(toggleValue(["a"], "b")).toEqual(["a", "b"]);
    expect(toggleValue(["a", "b"], "a")).toEqual(["b"]);
  });
  it("parses realtime frames and backs off", () => {
    expect(parseRealtimeEvent("not json")).toBeNull();
    expect(parseRealtimeEvent('{"type":"pong"}')).toEqual({ type: "pong" });
    expect(backoffDelay(0)).toBe(1000);
    expect(backoffDelay(10)).toBe(30_000);
  });
});
