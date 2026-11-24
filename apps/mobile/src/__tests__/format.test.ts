import { distanceLabel, relativeTime, shortTimestamp } from "../utils/format";

describe("format helpers", () => {
  const now = new Date(2026, 10, 24, 18, 30);

  it("never shows a precise distance under 2 km", () => {
    expect(distanceLabel(1)).toBe("À moins de 2 km");
    expect(distanceLabel(25)).toBe("À 25 km");
    expect(distanceLabel(null)).toBeNull();
  });

  it("formats conversation timestamps", () => {
    expect(shortTimestamp(new Date(2026, 10, 24, 9, 5).toISOString(), now)).toBe("09:05");
    expect(shortTimestamp(new Date(2026, 10, 23, 9, 5).toISOString(), now)).toBe("Hier");
    expect(shortTimestamp(new Date(2026, 9, 2, 9, 5).toISOString(), now)).toBe("02/10");
  });

  it("formats relative times", () => {
    expect(relativeTime(new Date(now.getTime() - 20_000).toISOString(), now)).toBe("À l'instant");
    expect(relativeTime(new Date(now.getTime() - 5 * 60_000).toISOString(), now)).toBe("Il y a 5 min");
  });
});
