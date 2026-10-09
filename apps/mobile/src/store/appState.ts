import type { ProfileSummary } from '../api/types';
import { createStore } from './simpleStore';

/** Conversation currently on screen (so realtime events do not toast / count it as unread). */
export const activeConversation = createStore<string | null>(null);

/** Bumped whenever the discovery deck must be rebuilt (preferences changed, unblock...). */
export const discoverEpoch = createStore<number>(0);
export const bumpDiscoverEpoch = () => discoverEpoch.set(discoverEpoch.get() + 1);

export type MatchCelebration = {
  matchId: string;
  conversationId: string;
  user: Pick<ProfileSummary, 'id' | 'firstName' | 'mainPhotoUrl'>;
};
export const matchCelebration = createStore<MatchCelebration | null>(null);
