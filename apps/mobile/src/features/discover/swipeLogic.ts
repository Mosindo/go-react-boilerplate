import type { SwipeAction } from '../../api/types';

/** Fraction of the card width a drag must cross to commit a swipe. */
export const SWIPE_DISTANCE_RATIO = 0.28;
/** A quick flick commits earlier (px/ms), as long as it travelled a minimum distance. */
export const SWIPE_VELOCITY = 0.55;
export const FLICK_MIN_DISTANCE_RATIO = 0.1;
export const MAX_ROTATION_DEG = 14;

/** Decides what a released drag means. `null` = snap back. */
export function decideSwipe(dx: number, vx: number, width: number): SwipeAction | null {
  if (width <= 0) return null;
  const far = Math.abs(dx) >= width * SWIPE_DISTANCE_RATIO;
  const flick =
    Math.abs(vx) >= SWIPE_VELOCITY &&
    Math.abs(dx) >= width * FLICK_MIN_DISTANCE_RATIO &&
    Math.sign(vx) === Math.sign(dx);
  if (!far && !flick) return null;
  return dx > 0 ? 'like' : 'pass';
}

export function rotationDeg(dx: number, width: number): number {
  if (width <= 0) return 0;
  const raw = (dx / width) * MAX_ROTATION_DEG * 2;
  return Math.max(-MAX_ROTATION_DEG, Math.min(MAX_ROTATION_DEG, raw));
}
