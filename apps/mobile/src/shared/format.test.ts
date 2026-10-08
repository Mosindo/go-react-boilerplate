import { formatDay, formatDistance, formatRelative, sameDay } from "./format";

const NOW = new Date("2026-03-15T12:00:00");

describe("formatRelative", () => {
  it("uses compact units", () => {
    expect(formatRelative("2026-03-15T11:59:50", NOW)).toBe("à l'instant");
    expect(formatRelative("2026-03-15T11:30:00", NOW)).toBe("30 min");
    expect(formatRelative("2026-03-15T09:00:00", NOW)).toBe("3 h");
    expect(formatRelative("2026-03-14T09:00:00", NOW)).toBe("hier");
    expect(formatRelative("garbage", NOW)).toBe("");
  });
});

describe("day helpers", () => {
  it("labels days", () => {
    expect(formatDay("2026-03-15T01:00:00", NOW)).toBe("Aujourd'hui");
    expect(formatDay("2026-03-14T23:00:00", NOW)).toBe("Hier");
    expect(sameDay("2026-03-15T01:00:00", "2026-03-15T23:00:00")).toBe(true);
    expect(sameDay("2026-03-15T01:00:00", "2026-03-16T01:00:00")).toBe(false);
  });
  it("describes distance coarsely", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(5)).toBe("À moins de 5 km");
    expect(formatDistance(35)).toBe("À 35 km");
  });
});
