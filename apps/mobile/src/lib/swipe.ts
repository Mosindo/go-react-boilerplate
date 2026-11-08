export type SwipeDecision = "like" | "pass" | null;

export const SWIPE_DISTANCE = 110;
export const SWIPE_VELOCITY = 0.9;

/** Decides what a drag gesture means. Horizontal intent only: vertical drags never swipe. */
export function decideSwipe(dx: number, dy: number, vx: number): SwipeDecision {
  if (Math.abs(dy) > Math.abs(dx) * 1.5) return null;
  if (dx > SWIPE_DISTANCE || (dx > 40 && vx > SWIPE_VELOCITY)) return "like";
  if (dx < -SWIPE_DISTANCE || (dx < -40 && vx < -SWIPE_VELOCITY)) return "pass";
  return null;
}
