import type { SwipeAction } from "../types";

export type GestureSample = { dx: number; dy: number; vx: number };
export type SwipeDecision = SwipeAction | "none";

/** Fraction of the screen width a drag must cover to commit a swipe. */
export const SWIPE_DISTANCE_RATIO = 0.3;
/** Horizontal velocity (px/ms) that commits a swipe if the card already moved a little. */
export const SWIPE_VELOCITY = 0.8;
export const SWIPE_MIN_FLING_DISTANCE = 40;
export const MAX_ROTATION_DEG = 12;

export function swipeThreshold(screenWidth: number): number {
  return screenWidth * SWIPE_DISTANCE_RATIO;
}

/** Decide what a finished drag means. Mostly-vertical drags never commit. */
export function decideSwipe(sample: GestureSample, screenWidth: number): SwipeDecision {
  const { dx, dy, vx } = sample;
  if (Math.abs(dy) > Math.abs(dx) * 1.5 && Math.abs(dx) < swipeThreshold(screenWidth)) {
    return "none";
  }
  const direction: SwipeAction | null = dx > 0 ? "like" : dx < 0 ? "pass" : null;
  if (direction === null) {
    return "none";
  }
  if (Math.abs(dx) >= swipeThreshold(screenWidth)) {
    return direction;
  }
  const flingsRight = vx >= SWIPE_VELOCITY && dx >= SWIPE_MIN_FLING_DISTANCE;
  const flingsLeft = vx <= -SWIPE_VELOCITY && dx <= -SWIPE_MIN_FLING_DISTANCE;
  if (flingsRight) {
    return "like";
  }
  if (flingsLeft) {
    return "pass";
  }
  return "none";
}

function clamp01(value: number): number {
  return Math.min(1, Math.max(0, value));
}

/** Opacity (0..1) of the LIKE / NOPE stamps while dragging. */
export function stampOpacity(dx: number, screenWidth: number): { like: number; pass: number } {
  const range = swipeThreshold(screenWidth);
  if (range <= 0) {
    return { like: 0, pass: 0 };
  }
  return { like: clamp01(dx / range), pass: clamp01(-dx / range) };
}

/** Card tilt in degrees for a horizontal drag. */
export function rotationFor(dx: number, screenWidth: number): number {
  if (screenWidth <= 0) {
    return 0;
  }
  const ratio = Math.min(1, Math.max(-1, dx / screenWidth));
  return ratio * MAX_ROTATION_DEG * 2;
}

/** Where the card flies to when a swipe is committed. */
export function exitTarget(action: SwipeAction, screenWidth: number): number {
  return (action === "like" ? 1 : -1) * screenWidth * 1.4;
}

/** True when the deck should request another batch. */
export function shouldPrefetch(remaining: number, hasMore: boolean, isFetching: boolean): boolean {
  return hasMore && !isFetching && remaining < 3;
}

/** Appends a batch while skipping profiles already in the deck or already acted on. */
export function appendBatch<T extends { userId: string }>(
  deck: T[],
  batch: T[],
  seen: ReadonlySet<string>
): T[] {
  const present = new Set(deck.map((p) => p.userId));
  const fresh: T[] = [];
  for (const item of batch) {
    if (present.has(item.userId) || seen.has(item.userId)) {
      continue;
    }
    present.add(item.userId);
    fresh.push(item);
  }
  return fresh.length === 0 ? deck : [...deck, ...fresh];
}

/** Photo carousel index after tapping the left (-1) or right (+1) half. */
export function nextPhotoIndex(current: number, total: number, step: -1 | 1): number {
  if (total <= 0) {
    return 0;
  }
  return Math.min(total - 1, Math.max(0, current + step));
}

/** Tap on left 40% -> previous photo, otherwise next. */
export function tapStep(x: number, width: number): -1 | 1 {
  return x < width * 0.4 ? -1 : 1;
}
