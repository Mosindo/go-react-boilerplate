import { formatDayLabel, formatDistance, formatListTime, sameDay } from "../lib/format";

describe("formatDistance", () => {
  it("never shows a precise distance", () => {
    expect(formatDistance(null)).toBeNull();
    expect(formatDistance(1)).toBe("À moins de 1 km");
    expect(formatDistance(25)).toBe("À moins de 25 km");
    expect(formatDistance(1000)).toBe("Très loin");
  });
});

describe("dates", () => {
  const now = new Date(2026, 5, 15, 12, 0, 0);

  it("labels today and yesterday", () => {
    expect(formatDayLabel(new Date(2026, 5, 15, 8, 0).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 5, 14, 23, 0).toISOString(), now)).toBe("Hier");
  });

  it("uses a clock time for messages from today in lists", () => {
    expect(formatListTime(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toMatch(/09[:h]05/);
  });

  it("compares days in local time", () => {
    expect(sameDay(new Date(2026, 5, 15, 1), new Date(2026, 5, 15, 23))).toBe(true);
    expect(sameDay(new Date(2026, 5, 15), new Date(2026, 5, 16))).toBe(false);
  });
});
