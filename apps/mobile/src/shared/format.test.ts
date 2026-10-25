import { dayKey, formatDayLabel, formatListTimestamp } from "./format";

const now = new Date(2026, 2, 12, 18, 0);

describe("format", () => {
  it("labels days relative to now", () => {
    expect(formatDayLabel(new Date(2026, 2, 12, 8).toISOString(), now)).toBe("Aujourd'hui");
    expect(formatDayLabel(new Date(2026, 2, 11, 8).toISOString(), now)).toBe("Hier");
    expect(formatDayLabel(new Date(2026, 0, 5, 8).toISOString(), now)).toBe("5 janvier");
    expect(formatDayLabel(new Date(2025, 0, 5, 8).toISOString(), now)).toBe("5 janvier 2025");
  });

  it("formats list timestamps", () => {
    expect(formatListTimestamp(new Date(2026, 2, 11, 8).toISOString(), now)).toBe("Hier");
    expect(formatListTimestamp(new Date(2026, 2, 3, 8).toISOString(), now)).toBe("03/03");
  });

  it("groups by local day", () => {
    expect(dayKey(new Date(2026, 2, 12, 1).toISOString())).toBe(dayKey(new Date(2026, 2, 12, 23).toISOString()));
  });
});
