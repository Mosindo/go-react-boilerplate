import { describe, expect, it } from "vitest";
import {
  dayKey,
  formatClock,
  formatDayLabel,
  formatDistance,
  formatNameAge,
  formatPlace,
  formatRelativeTime,
  humanizeSlug,
  previewText
} from "./format";

const now = new Date(2026, 5, 15, 12, 0, 0);
const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();
const MIN = 60_000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;

describe("formatRelativeTime", () => {
  it("formats by magnitude", () => {
    expect(formatRelativeTime(ago(10_000), now)).toBe("now");
    expect(formatRelativeTime(ago(5 * MIN), now)).toBe("5m");
    expect(formatRelativeTime(ago(3 * HOUR), now)).toBe("3h");
    expect(formatRelativeTime(ago(2 * DAY), now)).toBe("2d");
  });
  it("falls back to a date after a week", () => {
    expect(formatRelativeTime(new Date(2026, 5, 1, 10).toISOString(), now)).toBe("1 Jun");
    expect(formatRelativeTime(new Date(2025, 11, 24, 10).toISOString(), now)).toBe("24 Dec 2025");
  });
  it("handles garbage", () => {
    expect(formatRelativeTime("nope", now)).toBe("");
  });
});

describe("formatClock", () => {
  it("pads HH:mm", () => {
    expect(formatClock(new Date(2026, 0, 2, 7, 5).toISOString())).toBe("07:05");
    expect(formatClock(new Date(2026, 0, 2, 23, 45).toISOString())).toBe("23:45");
  });
});

describe("formatDayLabel", () => {
  it("labels today, yesterday, weekdays and dates", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 1).toISOString(), now)).toBe("Today");
    expect(formatDayLabel(new Date(2026, 5, 14, 23).toISOString(), now)).toBe("Yesterday");
    expect(formatDayLabel(new Date(2026, 5, 11, 9).toISOString(), now)).toBe("Thursday");
    expect(formatDayLabel(new Date(2026, 4, 1, 9).toISOString(), now)).toBe("1 May");
  });
  it("builds stable day keys", () => {
    expect(dayKey(new Date(2026, 0, 2, 7).toISOString())).toBe("2026-01-02");
  });
});

describe("distance and names", () => {
  it("formats distance", () => {
    expect(formatDistance(5)).toBe("about 5 km away");
    expect(formatDistance(0)).toBe("about 1 km away");
    expect(formatDistance(null)).toBeNull();
  });
  it("formats name, age and place", () => {
    expect(formatNameAge("Sam", 29)).toBe("Sam, 29");
    expect(formatNameAge("Sam", null)).toBe("Sam");
    expect(formatPlace("Lyon", 5)).toBe("Lyon · about 5 km away");
    expect(formatPlace("", 5)).toBe("about 5 km away");
    expect(formatPlace("Lyon", null)).toBe("Lyon");
  });
  it("humanizes slugs and previews", () => {
    expect(humanizeSlug("hiking_trips")).toBe("Hiking trips");
    expect(previewText("a\n\nb   c", 80)).toBe("a b c");
    expect(previewText("abcdefghij", 5)).toBe("abcd…");
  });
});
