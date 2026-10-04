export type SwipeDirection = "like" | "pass";

/** Decides whether a drag ended in a swipe. `dx` px, `vx` px/ms, `width` of the card. */
export function decideSwipe(dx: number, vx: number, width: number): SwipeDirection | null {
  const distanceThreshold = width * 0.28;
  const velocityThreshold = 0.6;
  if (dx > distanceThreshold || (dx > 40 && vx > velocityThreshold)) return "like";
  if (dx < -distanceThreshold || (dx < -40 && vx < -velocityThreshold)) return "pass";
  return null;
}

/** Which third of the card a tap landed in: previous photo, open profile, or next photo. */
export function tapZone(x: number, width: number): "prev" | "next" {
  return x < width / 2 ? "prev" : "next";
}

export function nextPhotoIndex(current: number, count: number, zone: "prev" | "next"): number {
  if (count <= 1) return 0;
  if (zone === "next") return Math.min(count - 1, current + 1);
  return Math.max(0, current - 1);
}
