import { formatDayLabel, formatDistance, formatRelative, sameDay } from "./format";

describe("format", () => {
  const now = new Date(2026, 2, 10, 15, 0, 0);

  it("formats relative times", () => {
    expect(formatRelative(new Date(2026, 2, 10, 14, 59, 30).toISOString(), now)).toBe("à l'instant");
    expect(formatRelative(new Date(2026, 2, 10, 14, 50).toISOString(), now)).toBe("10 min");
    expect(formatRelative(new Date(2026, 2, 9, 9, 0).toISOString(), now)).toBe("hier");
  });

  it("labels chat days", () => {
    expect(formatDayLabel(new Date(2026, 2, 10, 8).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 2, 9, 8).toISOString(), now)).toBe("Hier");
    expect(sameDay(new Date(2026, 2, 10, 1).toISOString(), new Date(2026, 2, 10, 23).toISOString())).toBe(true);
  });

  it("never shows a precise distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("à moins d'1 km");
    expect(formatDistance(15)).toBe("à 15 km");
  });
});
