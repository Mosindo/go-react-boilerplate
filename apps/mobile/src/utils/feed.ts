import type { Card } from "../api/types";

/**
 * Keeps the order of cards already on screen when the feed is refreshed, so
 * the card under the user's finger never changes. Profiles that are no longer
 * eligible disappear; new ones are appended.
 */
export function mergeFeed(previous: Card[] | undefined, fresh: Card[]): Card[] {
  if (!previous || previous.length === 0) {
    return fresh;
  }
  const freshById = new Map(fresh.map((c) => [c.userId, c]));
  const kept = previous.filter((c) => freshById.has(c.userId)).map((c) => freshById.get(c.userId) as Card);
  const keptIds = new Set(kept.map((c) => c.userId));
  return [...kept, ...fresh.filter((c) => !keptIds.has(c.userId))];
}
