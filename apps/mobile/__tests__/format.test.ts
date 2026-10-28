import { distanceLabel, timeLabel, errorMessage } from "../src/lib/format";

describe("distanceLabel", () => {
  it("hides missing distances", () => {
    expect(distanceLabel(null)).toBeNull();
    expect(distanceLabel(undefined)).toBeNull();
  });
  it("is approximate", () => {
    expect(distanceLabel(1)).toBe("Less than 1 km away");
    expect(distanceLabel(5)).toBe("About 5 km away");
    expect(distanceLabel(14.6)).toBe("About 15 km away");
  });
});

describe("timeLabel", () => {
  const now = new Date(2026, 5, 15, 12, 0);
  it("shows the clock for today", () => {
    expect(timeLabel(new Date(2026, 5, 15, 9, 5).toISOString(), now)).toBe("09:05");
  });
  it("shows the weekday within a week and the date after", () => {
    expect(timeLabel(new Date(2026, 5, 12, 9, 5).toISOString(), now)).toBe("Fri");
    expect(timeLabel(new Date(2026, 4, 1, 9, 5).toISOString(), now)).toBe("01/05");
  });
  it("tolerates garbage", () => {
    expect(timeLabel("nope", now)).toBe("");
  });
});

describe("errorMessage", () => {
  it("uses Error messages, else the fallback", () => {
    expect(errorMessage(new Error("boom"))).toBe("boom");
    expect(errorMessage(42, "fallback")).toBe("fallback");
  });
});
