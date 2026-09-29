import { describe, expect, it } from "vitest";
import {
  canAddPhoto,
  guessImageMime,
  isPermutation,
  isTooLarge,
  makeMain,
  movePhoto,
  photoIds,
  remainingSlots,
  sameOrder,
  sortPhotos,
  uploadFileName
} from "./photos";

describe("photo ordering", () => {
  const photos = [
    { id: "c", position: 2 },
    { id: "a", position: 0 },
    { id: "b", position: 1 }
  ];
  it("sorts by position without mutating", () => {
    expect(sortPhotos(photos).map((p) => p.id)).toEqual(["a", "b", "c"]);
    expect(photos[0]?.id).toBe("c");
    expect(photoIds(photos)).toEqual(["a", "b", "c"]);
  });
  it("moves photos and ignores out-of-range moves", () => {
    expect(movePhoto(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
    expect(movePhoto(["a", "b", "c"], 0, 1)).toEqual(["b", "a", "c"]);
    expect(movePhoto(["a", "b", "c"], 0, 5)).toEqual(["a", "b", "c"]);
    expect(movePhoto(["a", "b", "c"], -1, 1)).toEqual(["a", "b", "c"]);
  });
  it("makes a photo main", () => {
    expect(makeMain(["a", "b", "c"], "c")).toEqual(["c", "a", "b"]);
    expect(makeMain(["a", "b", "c"], "zzz")).toEqual(["a", "b", "c"]);
  });
  it("compares orders", () => {
    expect(isPermutation(["a", "b"], ["b", "a"])).toBe(true);
    expect(isPermutation(["a", "b"], ["b", "b"])).toBe(false);
    expect(isPermutation(["a"], ["a", "b"])).toBe(false);
    expect(sameOrder(["a", "b"], ["a", "b"])).toBe(true);
    expect(sameOrder(["a", "b"], ["b", "a"])).toBe(false);
  });
  it("tracks the 6 photo limit", () => {
    expect(canAddPhoto(5)).toBe(true);
    expect(canAddPhoto(6)).toBe(false);
    expect(remainingSlots(4)).toBe(2);
    expect(remainingSlots(9)).toBe(0);
  });
});

describe("upload metadata", () => {
  it("guesses mime types", () => {
    expect(guessImageMime("file:///a/b.PNG")).toBe("image/png");
    expect(guessImageMime("file:///a/b.webp?x=1")).toBe("image/webp");
    expect(guessImageMime("file:///a/b.heic")).toBe("image/jpeg");
    expect(guessImageMime("file:///a/noext", "image/png")).toBe("image/png");
    expect(guessImageMime("file:///a/noext", "application/pdf")).toBe("image/jpeg");
  });
  it("derives a file name", () => {
    expect(uploadFileName("file:///a.jpg", "me.jpeg")).toBe("me.jpeg");
    expect(uploadFileName("file:///a.png", null, 3)).toMatch(/^photo-\d+-3\.png$/);
  });
  it("flags files over 5 MB", () => {
    expect(isTooLarge(5 * 1024 * 1024)).toBe(false);
    expect(isTooLarge(5 * 1024 * 1024 + 1)).toBe(true);
    expect(isTooLarge(undefined)).toBe(false);
  });
});
