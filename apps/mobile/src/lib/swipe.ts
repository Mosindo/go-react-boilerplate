import type { SwipeAction } from "../api/types";

export type SwipeGesture = { dx: number; dy: number; vx: number };

export const SWIPE_DISTANCE_RATIO = 0.28;
export const SWIPE_MIN_DISTANCE = 90;
export const SWIPE_VELOCITY = 0.6;
export const TAP_SLOP = 8;

export function swipeThreshold(containerWidth: number): number {
  return Math.max(SWIPE_MIN_DISTANCE, containerWidth * SWIPE_DISTANCE_RATIO);
}

/**
 * Decides what a released drag means. Right = like, left = pass.
 * A fast flick counts when the card moved a little in the same direction;
 * mostly vertical drags cancel.
 */
export function decideSwipe(gesture: SwipeGesture, containerWidth: number): SwipeAction | null {
  const { dx, dy, vx } = gesture;
  const threshold = swipeThreshold(containerWidth);
  if (Math.abs(dy) > Math.abs(dx) * 1.5 && Math.abs(dx) < threshold) return null;
  if (dx >= threshold) return "like";
  if (dx <= -threshold) return "pass";
  if (Math.abs(vx) >= SWIPE_VELOCITY && Math.abs(dx) > 40) return dx > 0 ? "like" : "pass";
  return null;
}

/** True when the touch was a tap rather than a drag. */
export function isTap(gesture: SwipeGesture): boolean {
  return Math.abs(gesture.dx) < TAP_SLOP && Math.abs(gesture.dy) < TAP_SLOP;
}

/** Opacity (0..1) of the like/pass stamp for the current drag distance. */
export function stampOpacity(dx: number, action: SwipeAction, containerWidth: number): number {
  const signed = action === "like" ? dx : -dx;
  if (signed <= 0) return 0;
  return Math.min(1, signed / swipeThreshold(containerWidth));
}

/** Card rotation in degrees for a horizontal drag. */
export function rotationDegrees(dx: number, containerWidth: number): number {
  const ratio = Math.max(-1, Math.min(1, dx / Math.max(1, containerWidth)));
  return ratio * 12;
}

/** Cards still to show: server list minus anything swiped locally (the server replays unswiped cards). */
export function remainingDeck<T extends { userId: string }>(
  cards: T[],
  swiped: ReadonlySet<string>
): T[] {
  return cards.filter((card) => !swiped.has(card.userId));
}

export function shouldPrefetch(remaining: number, threshold = 3): boolean {
  return remaining <= threshold;
}
