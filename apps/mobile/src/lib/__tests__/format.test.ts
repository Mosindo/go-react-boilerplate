import { describe, expect, it } from "vitest";
import { formatDayLabel, formatDistance, formatRelativeTime, pluralize } from "../format";
import { searchCities } from "../cities";
import { mergeQueue, needsRefill } from "../swipeQueue";

const now = new Date("2026-06-15T12:00:00Z");

describe("format", () => {
  it("describes distance without false precision", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 1 km");
    expect(formatDistance(15)).toBe("À 15 km");
  });

  it("formats relative times", () => {
    expect(formatRelativeTime("2026-06-15T11:59:50Z", now)).toBe("À l'instant");
    expect(formatRelativeTime("2026-06-15T11:30:00Z", now)).toBe("Il y a 30 min");
    expect(formatRelativeTime("2026-06-15T09:00:00Z", now)).toBe("Il y a 3 h");
    expect(formatRelativeTime("2026-06-13T12:00:00Z", now)).toBe("Il y a 2 j");
    expect(formatRelativeTime("garbage", now)).toBe("");
  });

  it("labels today and yesterday", () => {
    expect(formatDayLabel("2026-06-15T08:00:00Z", now)).toBe("Aujourd'hui");
    expect(formatDayLabel("2026-06-14T08:00:00Z", now)).toBe("Hier");
  });

  it("pluralizes", () => {
    expect(pluralize(1, "match", "matchs")).toBe("match");
    expect(pluralize(2, "match", "matchs")).toBe("matchs");
  });
});

describe("cities", () => {
  it("searches ignoring accents and case", () => {
    expect(searchCities("geneve").map((c) => c.name)).toEqual(["Genève"]);
    expect(searchCities("  PARIS ").map((c) => c.name)).toEqual(["Paris"]);
    expect(searchCities("").length).toBeGreaterThan(10);
    expect(searchCities("zzz")).toHaveLength(0);
  });
});

describe("swipe queue", () => {
  it("dedupes against the queue and already swiped ids", () => {
    const merged = mergeQueue([{ id: "a" }, { id: "b" }], [{ id: "b" }, { id: "c" }, { id: "d" }], new Set(["d"]));
    expect(merged.map((p) => p.id)).toEqual(["a", "b", "c"]);
  });
  it("asks for more when the deck runs low", () => {
    expect(needsRefill(2)).toBe(true);
    expect(needsRefill(8)).toBe(false);
  });
});
