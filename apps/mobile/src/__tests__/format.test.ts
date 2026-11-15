import { formatDayHeader, formatDistance, formatListDate, relativeTime } from "../lib/format";

describe("format", () => {
  const now = new Date(2026, 2, 12, 15, 0, 0);

  it("never shows a precise distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 2 km");
    expect(formatDistance(14)).toBe("À 14 km");
  });

  it("formats list dates", () => {
    expect(formatListDate(new Date(2026, 2, 12, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListDate(new Date(2026, 2, 11, 22, 0).toISOString(), now)).toBe("Hier");
    expect(formatListDate(new Date(2026, 0, 3).toISOString(), now)).toBe("03/01");
  });

  it("formats day headers and relative times", () => {
    expect(formatDayHeader(new Date(2026, 2, 12, 1).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayHeader(new Date(2025, 11, 25).toISOString(), now)).toBe("25 décembre 2025");
    expect(relativeTime(new Date(2026, 2, 12, 14, 58).toISOString(), now)).toBe("Il y a 2 min");
    expect(relativeTime(new Date(2026, 2, 12, 14, 59, 50).toISOString(), now)).toBe("À l'instant");
  });
});
