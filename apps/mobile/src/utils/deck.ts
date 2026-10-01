import type { PublicProfile } from "../api/types";

export const DECK_REFILL_THRESHOLD = 3;

/**
 * Appends freshly fetched candidates without duplicates and without anything
 * already swiped in this session (the server also excludes swiped profiles;
 * this protects against in-flight swipes).
 */
export function mergeCandidates(queue: PublicProfile[], incoming: PublicProfile[], swiped: ReadonlySet<string>): PublicProfile[] {
  const known = new Set(queue.map((p) => p.id));
  const fresh = incoming.filter((p) => !known.has(p.id) && !swiped.has(p.id));
  return fresh.length === 0 ? queue : [...queue, ...fresh];
}

/**
 * Result of a refill: the new queue, and whether the server has nothing more
 * to offer. A batch that adds nobody new (empty, or only profiles we already
 * hold / just swiped) must stop the refill loop, otherwise a short candidate
 * list would be refetched forever.
 */
export function applyBatch(
  queue: PublicProfile[],
  incoming: PublicProfile[],
  swiped: ReadonlySet<string>
): { queue: PublicProfile[]; exhausted: boolean } {
  const merged = mergeCandidates(queue, incoming, swiped);
  return { queue: merged, exhausted: merged === queue };
}

export function needsRefill(queueLength: number, isFetching: boolean, exhausted: boolean): boolean {
  return !isFetching && !exhausted && queueLength <= DECK_REFILL_THRESHOLD;
}

/** Swipe distance (px) after which a release commits to like / pass. */
export function swipeDecision(dx: number, vx: number, width: number): "like" | "pass" | null {
  const threshold = width * 0.28;
  if (dx > threshold || (dx > 40 && vx > 0.9)) {
    return "like";
  }
  if (dx < -threshold || (dx < -40 && vx < -0.9)) {
    return "pass";
  }
  return null;
}
