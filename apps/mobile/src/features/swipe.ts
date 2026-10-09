import type { QueryClient } from '@tanstack/react-query';
import { discoverApi } from '../api/discover';
import type { ProfileSummary, PublicProfile, SwipeAction, SwipeResult } from '../api/types';
import { qk } from '../query/keys';
import { toast } from '../shared/toastStore';
import { matchCelebration } from '../store/appState';
import { createStore } from '../store/simpleStore';

export type SwipeTarget = Pick<ProfileSummary, 'id' | 'firstName' | 'mainPhotoUrl'>;

export function toSwipeTarget(p: PublicProfile): SwipeTarget {
  return { id: p.id, firstName: p.firstName, mainPhotoUrl: p.photos[0]?.url ?? null };
}

/** Lets the profile screen tell the Discover deck that a card was already handled. */
export const swipeBus = createStore<{ userId: string; action: SwipeAction; n: number } | null>(
  null,
);
let busCounter = 0;
export function announceSwipe(userId: string, action: SwipeAction): void {
  busCounter += 1;
  swipeBus.set({ userId, action, n: busCounter });
}

/** POST /swipes + all the side effects shared by Discover, Likes and the profile screen. */
export async function performSwipe(
  qc: QueryClient,
  target: SwipeTarget,
  action: SwipeAction,
): Promise<SwipeResult> {
  const result = await discoverApi.swipe(target.id, action);
  // They can no longer appear in "who liked me" once swiped.
  qc.setQueryData<{ pages: PublicProfile[][]; pageParams: unknown[] }>(qk.likes, (old) =>
    old ? { ...old, pages: old.pages.map((page) => page.filter((p) => p.id !== target.id)) } : old,
  );
  if (result.matched && result.match) {
    toast.clear(); // a realtime "new match" toast may have beaten the HTTP response
    matchCelebration.set({
      matchId: result.match.id,
      conversationId: result.match.conversationId,
      user: target,
    });
    void qc.invalidateQueries({ queryKey: qk.conversations });
    void qc.invalidateQueries({ queryKey: qk.matches });
    void qc.invalidateQueries({ queryKey: qk.notifications });
  }
  return result;
}
