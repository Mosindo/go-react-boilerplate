import { decideSwipe, nextPhotoIndex, photoTapDirection } from "../src/lib/swipe";

describe("decideSwipe", () => {
  const width = 400;
  it("commits past the distance threshold", () => {
    expect(decideSwipe(120, 0, width)).toBe("like");
    expect(decideSwipe(-120, 0, width)).toBe("pass");
  });
  it("snaps back for small slow drags", () => {
    expect(decideSwipe(30, 0.1, width)).toBeNull();
    expect(decideSwipe(-60, 0.2, width)).toBeNull();
  });
  it("commits on a fast fling beyond the minimum distance", () => {
    expect(decideSwipe(60, 1.2, width)).toBe("like");
    expect(decideSwipe(-60, -1.2, width)).toBe("pass");
    expect(decideSwipe(10, 2, width)).toBeNull();
  });
});

describe("photo navigation", () => {
  it("maps tap zones", () => {
    expect(photoTapDirection(10, 300)).toBe("prev");
    expect(photoTapDirection(250, 300)).toBe("next");
  });
  it("clamps indices", () => {
    expect(nextPhotoIndex(0, "prev", 3)).toBe(0);
    expect(nextPhotoIndex(2, "next", 3)).toBe(2);
    expect(nextPhotoIndex(1, "next", 3)).toBe(2);
    expect(nextPhotoIndex(0, "next", 0)).toBe(0);
  });
});
