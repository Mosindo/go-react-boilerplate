import type { SwipeAction } from "../api/types";

export const SWIPE_DISTANCE_RATIO = 0.28;
export const SWIPE_VELOCITY = 0.9;
export const SWIPE_MIN_FLING_DISTANCE = 40;

/** Decides whether a released drag commits to a like/pass or snaps back. */
export function decideSwipe(dx: number, vx: number, width: number): SwipeAction | null {
  if (Math.abs(dx) >= width * SWIPE_DISTANCE_RATIO) {
    return dx > 0 ? "like" : "pass";
  }
  if (Math.abs(vx) >= SWIPE_VELOCITY && Math.abs(dx) >= SWIPE_MIN_FLING_DISTANCE) {
    return dx > 0 ? "like" : "pass";
  }
  return null;
}

/** Tap zones on a photo: left third = previous, otherwise next. */
export function photoTapDirection(x: number, width: number): "prev" | "next" {
  return x < width / 3 ? "prev" : "next";
}

export function nextPhotoIndex(current: number, direction: "prev" | "next", count: number): number {
  if (count <= 0) return 0;
  const next = direction === "next" ? current + 1 : current - 1;
  return Math.min(Math.max(next, 0), count - 1);
}
