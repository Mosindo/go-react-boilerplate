/**
 * The server pushes `match.new` to both people, including the one whose swipe created the match.
 * That person already gets the match modal, so the realtime toast would be a duplicate. The push
 * can land before or after the swipe response: a swipe stays registered while in flight and, when
 * it produced a match, for a short grace period afterwards.
 */
const GRACE_MS = 10_000;

const swipes = new Map<string, number>();

/** Call before sending the swipe. */
export function swipeStarted(userId: string): void {
  swipes.set(userId, Number.POSITIVE_INFINITY);
}

/** Call once the swipe response (or failure) is known. */
export function swipeFinished(userId: string, matched: boolean): void {
  if (matched) swipes.set(userId, Date.now() + GRACE_MS);
  else swipes.delete(userId);
}

/** True when a match with `userId` was just created by this device's own swipe. */
export function isOwnSwipeMatch(userId: string, now: number = Date.now()): boolean {
  const until = swipes.get(userId);
  return until !== undefined && until > now;
}
