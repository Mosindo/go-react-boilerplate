import { computeResize, moveItem } from "../photos";

describe("computeResize", () => {
  it("leaves small images alone", () => {
    expect(computeResize(800, 600)).toBeNull();
    expect(computeResize(1280, 1280)).toBeNull();
  });

  it("caps the longest side at 1280", () => {
    expect(computeResize(4000, 3000)).toEqual({ width: 1280 });
    expect(computeResize(3000, 4000)).toEqual({ height: 1280 });
  });
});

describe("moveItem", () => {
  it("moves items and never mutates the input", () => {
    const input = ["a", "b", "c"];
    expect(moveItem(input, 2, -2)).toEqual(["c", "a", "b"]);
    expect(moveItem(input, 0, 1)).toEqual(["b", "a", "c"]);
    expect(input).toEqual(["a", "b", "c"]);
  });

  it("ignores moves that fall outside the list", () => {
    const input = ["a", "b"];
    expect(moveItem(input, 0, -1)).toBe(input);
    expect(moveItem(input, 1, 1)).toBe(input);
  });
});
