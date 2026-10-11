import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { discoverApi, queryKeys } from '../../api';
import type { PublicProfile, SwipeAction, SwipeResult } from '../../api';
import { useMatch } from './MatchContext';

/** POST /swipes; opens the match modal and refreshes conversations on a match. Throws on failure. */
export function useSwipe() {
  const qc = useQueryClient();
  const { showMatch } = useMatch();
  return useCallback(
    async (profile: PublicProfile, action: SwipeAction): Promise<SwipeResult> => {
      const result = await discoverApi.swipe(profile.userId, action);
      if (result.matched && result.conversation) {
        showMatch(result.conversation, profile);
        void qc.invalidateQueries({ queryKey: queryKeys.conversations });
        void qc.invalidateQueries({ queryKey: queryKeys.notifications });
      }
      return result;
    },
    [qc, showMatch],
  );
}
