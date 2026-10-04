import { decideSwipe, nextPhotoIndex, tapZone } from "../swipe";
import { moveId, validatePickedPhoto } from "../photos";
import type { Photo } from "../../api/types";

describe("decideSwipe", () => {
  it("swipes on distance or on a fast flick, otherwise returns to center", () => {
    expect(decideSwipe(150, 0, 400)).toBe("like");
    expect(decideSwipe(-150, 0, 400)).toBe("pass");
    expect(decideSwipe(60, 0.9, 400)).toBe("like");
    expect(decideSwipe(-60, -0.9, 400)).toBe("pass");
    expect(decideSwipe(60, 0.1, 400)).toBeNull();
    expect(decideSwipe(10, 2, 400)).toBeNull();
  });
});

describe("photo carousel taps", () => {
  it("picks left/right zones and clamps the index", () => {
    expect(tapZone(10, 300)).toBe("prev");
    expect(tapZone(250, 300)).toBe("next");
    expect(nextPhotoIndex(0, 3, "prev")).toBe(0);
    expect(nextPhotoIndex(2, 3, "next")).toBe(2);
    expect(nextPhotoIndex(1, 3, "next")).toBe(2);
    expect(nextPhotoIndex(0, 1, "next")).toBe(0);
  });
});

describe("photos", () => {
  const photos = ["a", "b", "c"].map((id, position): Photo => ({ id, position, url: "", thumbUrl: "" }));
  it("reorders ids", () => {
    expect(moveId(photos, 2, 0)).toEqual(["c", "a", "b"]);
    expect(moveId(photos, 0, 1)).toEqual(["b", "a", "c"]);
    expect(moveId(photos, 0, 9)).toEqual(["a", "b", "c"]);
  });
  it("mirrors server upload constraints", () => {
    expect(validatePickedPhoto({ mimeType: "image/jpeg", fileSize: 1000, width: 800, height: 600 })).toBeNull();
    expect(validatePickedPhoto({ mimeType: "image/gif" })).not.toBeNull();
    expect(validatePickedPhoto({ fileSize: 9 * 1024 * 1024 })).not.toBeNull();
    expect(validatePickedPhoto({ width: 9000, height: 100 })).not.toBeNull();
  });
});
