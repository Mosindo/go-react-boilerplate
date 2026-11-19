import { formatDayLabel, formatDistance, formatListDate } from "./format";

describe("format", () => {
  it("shows only approximate distances", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 2 km");
    expect(formatDistance(15)).toBe("À environ 15 km");
  });

  it("formats list dates relative to now", () => {
    const now = new Date(2026, 5, 15, 18, 0);
    expect(formatListDate(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
    expect(formatListDate(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
    expect(formatListDate(new Date(2026, 5, 11, 12, 0).toISOString(), now)).toBe("jeu.");
    expect(formatListDate(new Date(2026, 1, 3, 12, 0).toISOString(), now)).toBe("03/02");
  });

  it("formats day separators", () => {
    const now = new Date(2026, 5, 15, 18, 0);
    expect(formatDayLabel(new Date(2026, 5, 15, 8).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 5, 14, 8).toISOString(), now)).toBe("Hier");
    expect(formatDayLabel(new Date(2026, 2, 2, 8).toISOString(), now)).toBe("2 mars");
    expect(formatDayLabel(new Date(2025, 2, 2, 8).toISOString(), now)).toBe("2 mars 2025");
  });
});
