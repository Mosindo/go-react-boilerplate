import { formatDistance, formatListTime, formatRelative, isSameDay } from "./format";

describe("format", () => {
  const now = new Date(2026, 10, 25, 15, 0, 0);

  it("formats list timestamps", () => {
    expect(formatListTime(new Date(2026, 10, 25, 9, 5).toISOString(), now)).toMatch(/09:05/);
    expect(formatListTime(new Date(2026, 10, 24, 9, 5).toISOString(), now)).toBe("Hier");
  });

  it("formats approximate distances without precision", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 2 km");
    expect(formatDistance(15)).toBe("À 15 km");
  });

  it("formats relative times", () => {
    expect(formatRelative(new Date(now.getTime() - 20_000).toISOString(), now)).toBe("à l'instant");
    expect(formatRelative(new Date(now.getTime() - 5 * 60_000).toISOString(), now)).toBe("il y a 5 min");
    expect(formatRelative(new Date(now.getTime() - 3 * 3_600_000).toISOString(), now)).toBe("il y a 3 h");
  });

  it("compares days", () => {
    expect(isSameDay(new Date(2026, 0, 1, 1).toISOString(), new Date(2026, 0, 1, 23).toISOString())).toBe(true);
  });
});
