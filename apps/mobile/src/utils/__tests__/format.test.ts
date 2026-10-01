import { formatClock, formatDayLabel, formatDistance, formatRelativeTime, initialOf } from "../format";

const NOW = new Date(2026, 5, 15, 14, 0, 0);

describe("formatRelativeTime", () => {
  it("words recent times for conversation lists", () => {
    expect(formatRelativeTime(new Date(NOW.getTime() - 20_000).toISOString(), NOW)).toBe("À l'instant");
    expect(formatRelativeTime(new Date(NOW.getTime() - 5 * 60_000).toISOString(), NOW)).toBe("5 min");
    expect(formatRelativeTime(new Date(2026, 5, 15, 11, 0).toISOString(), NOW)).toBe("3 h");
    expect(formatRelativeTime(new Date(2026, 5, 14, 20, 0).toISOString(), NOW)).toBe("Hier");
    expect(formatRelativeTime(new Date(2026, 5, 1, 9, 0).toISOString(), NOW)).toBe("01/06");
  });
});

describe("formatDayLabel", () => {
  it("labels today, yesterday and older days", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 8).toISOString(), NOW)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 5, 14, 23).toISOString(), NOW)).toBe("Hier");
    expect(formatDayLabel(new Date(2026, 0, 2).toISOString(), NOW)).toBe("02/01/2026");
  });
});

describe("misc formatters", () => {
  it("formats the clock with leading zeros", () => {
    expect(formatClock(new Date(2026, 5, 15, 9, 5).toISOString())).toBe("09:05");
  });

  it("never invents precision for distances", () => {
    expect(formatDistance(null)).toBe("");
    expect(formatDistance(undefined)).toBe("");
    expect(formatDistance(1)).toBe("À moins de 1 km");
    expect(formatDistance(15)).toBe("À 15 km");
  });

  it("extracts an initial", () => {
    expect(initialOf("  léa")).toBe("L");
  });
});
