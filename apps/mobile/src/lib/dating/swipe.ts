export type SwipeAction = "like" | "pass";

/** Fraction of the screen width the card must travel before a release counts as a decision. */
export const SWIPE_DISTANCE_RATIO = 0.28;
/** Horizontal velocity (px/ms) that counts as a flick. */
export const SWIPE_VELOCITY = 0.6;
/** Minimum travel for a flick to count, so tiny fast jitters do not swipe. */
export const SWIPE_MIN_FLICK_DISTANCE = 40;
/** Fraction of the screen width at which the LIKE / PASS stamp is fully opaque. */
export const STAMP_FULL_RATIO = 0.25;

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Decide what a released drag means. Returns null when the card should spring back. */
export function decideSwipe(dx: number, vx: number, screenWidth: number): SwipeAction | null {
  const threshold = screenWidth * SWIPE_DISTANCE_RATIO;
  if (dx >= threshold) return "like";
  if (dx <= -threshold) return "pass";
  if (Math.abs(vx) >= SWIPE_VELOCITY && Math.abs(dx) >= SWIPE_MIN_FLICK_DISTANCE) {
    return vx > 0 ? "like" : "pass";
  }
  return null;
}

/** Opacity (0..1) of the LIKE stamp for a horizontal drag distance. */
export function likeStampOpacity(dx: number, screenWidth: number): number {
  return clamp(dx / (screenWidth * STAMP_FULL_RATIO), 0, 1);
}

/** Opacity (0..1) of the PASS stamp for a horizontal drag distance. */
export function passStampOpacity(dx: number, screenWidth: number): number {
  return clamp(-dx / (screenWidth * STAMP_FULL_RATIO), 0, 1);
}

/** Where a card should land when it flies off screen. */
export function flyOffTarget(action: SwipeAction, screenWidth: number): number {
  return (action === "like" ? 1 : -1) * screenWidth * 1.4;
}

export type SwipeFailure = "already-swiped" | "unavailable" | "profile-incomplete" | "retry";

/** Map an HTTP status (null for network failures) to how the deck must react. */
export function classifySwipeFailure(status: number | null): SwipeFailure {
  if (status === 409) return "already-swiped";
  if (status === 404) return "unavailable";
  if (status === 422) return "profile-incomplete";
  return "retry";
}
