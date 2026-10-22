import { distanceLabel, formatDayHeading, formatShortTime, genderLabel, isSameDay } from "./format";

const NOW = new Date(2026, 5, 15, 18, 0, 0);

describe("format", () => {
  it("shows the clock for today and 'Hier' for yesterday", () => {
    expect(formatShortTime(new Date(2026, 5, 15, 9, 5).toISOString(), NOW)).toMatch(/09[:h]05/);
    expect(formatShortTime(new Date(2026, 5, 14, 23, 0).toISOString(), NOW)).toBe("Hier");
  });

  it("labels days", () => {
    expect(formatDayHeading(new Date(2026, 5, 15, 1).toISOString(), NOW)).toBe("Aujourd'hui");
    expect(formatDayHeading(new Date(2026, 5, 14, 1).toISOString(), NOW)).toBe("Hier");
  });

  it("compares days", () => {
    expect(isSameDay(new Date(2026, 5, 15, 1).toISOString(), new Date(2026, 5, 15, 23).toISOString())).toBe(true);
    expect(isSameDay(new Date(2026, 5, 15, 1).toISOString(), new Date(2026, 5, 16, 1).toISOString())).toBe(false);
  });

  it("only exposes approximate distances", () => {
    expect(distanceLabel(undefined)).toBeNull();
    expect(distanceLabel(5)).toBe("À moins de 5 km");
    expect(distanceLabel(15)).toBe("À environ 15 km");
  });

  it("translates genders", () => {
    expect(genderLabel("non_binary")).toBe("Non-binaire");
  });
});
