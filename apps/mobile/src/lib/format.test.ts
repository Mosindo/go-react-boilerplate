import { formatDayLabel, formatDistance, formatRelative } from "./format";

const NOW = new Date(2026, 5, 15, 12, 0, 0);

describe("format helpers", () => {
  it("formats relative times", () => {
    expect(formatRelative(new Date(NOW.getTime() - 20_000).toISOString(), NOW)).toBe("à l'instant");
    expect(formatRelative(new Date(NOW.getTime() - 5 * 60_000).toISOString(), NOW)).toBe("5 min");
    expect(formatRelative(new Date(NOW.getTime() - 3 * 3_600_000).toISOString(), NOW)).toBe("3 h");
    expect(formatRelative(new Date(NOW.getTime() - 30 * 3_600_000).toISOString(), NOW)).toBe(
      "hier"
    );
  });

  it("labels chat days", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 8).toISOString(), NOW)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 5, 14, 23).toISOString(), NOW)).toBe("Hier");
  });

  it("formats approximate distances", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("à moins de 1 km");
    expect(formatDistance(15)).toBe("à 15 km");
  });
});
