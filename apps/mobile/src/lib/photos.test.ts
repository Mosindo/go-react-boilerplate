import { describe, expect, it } from "vitest";
import { moveId } from "./photos";

describe("moveId", () => {
  it("moves a photo to the front (set as main)", () => {
    expect(moveId(["a", "b", "c", "d"], 2, 0)).toEqual(["c", "a", "b", "d"]);
  });

  it("moves a photo to the end", () => {
    expect(moveId(["a", "b", "c"], 0, 2)).toEqual(["b", "c", "a"]);
  });

  it("ignores impossible moves", () => {
    const ids = ["a", "b"];
    expect(moveId(ids, 0, 0)).toBe(ids);
    expect(moveId(ids, -1, 1)).toBe(ids);
    expect(moveId(ids, 0, 5)).toBe(ids);
  });
});
