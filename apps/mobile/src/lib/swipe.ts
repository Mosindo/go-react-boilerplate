export type SwipeDecision = "like" | "pass" | null;

const DISTANCE_RATIO = 0.28;
const VELOCITY = 0.6;

/** Decides whether a drag ended as a like, a pass, or should snap back. */
export function decideSwipe(dx: number, vx: number, width: number): SwipeDecision {
  if (dx > width * DISTANCE_RATIO || (dx > 40 && vx > VELOCITY)) {
    return "like";
  }
  if (dx < -width * DISTANCE_RATIO || (dx < -40 && vx < -VELOCITY)) {
    return "pass";
  }
  return null;
}
