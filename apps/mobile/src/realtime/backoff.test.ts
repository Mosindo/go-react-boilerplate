import { describe, expect, it } from "vitest";
import { backoffDelay } from "./backoff";

describe("backoffDelay", () => {
  it("doubles each attempt and is capped", () => {
    const mid = () => 0.5; // jitter factor 1.0
    expect([0, 1, 2, 3].map((a) => backoffDelay(a, mid))).toEqual([1000, 2000, 4000, 8000]);
    expect(backoffDelay(20, mid)).toBe(30_000);
  });

  it("keeps jitter within +-25%", () => {
    expect(backoffDelay(2, () => 0)).toBe(3000);
    expect(backoffDelay(2, () => 1)).toBe(5000);
  });
});
