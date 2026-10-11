import { useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { discoverApi, queryKeys } from '../../api';
import type { PublicProfile, SwipeAction } from '../../api';
import { useErrorToast } from '../../lib/useErrorToast';
import { useDeck } from '../discover/DeckContext';
import { useSwipe } from '../discover/useSwipe';

export function useLikes() {
  return useQuery({
    queryKey: queryKeys.likes,
    queryFn: () => discoverApi.likesReceived(),
    staleTime: 30_000,
  });
}

/** Like back / pass on someone who liked you: optimistic removal, rollback on failure. */
export function useRespondToLike() {
  const qc = useQueryClient();
  const swipe = useSwipe();
  const deck = useDeck();
  const onError = useErrorToast();
  return useCallback(
    (profile: PublicProfile, action: SwipeAction) => {
      qc.setQueryData<PublicProfile[]>(queryKeys.likes, (old) =>
        old?.filter((p) => p.userId !== profile.userId),
      );
      deck.remove(profile.userId);
      swipe(profile, action).catch((error: unknown) => {
        void qc.invalidateQueries({ queryKey: queryKeys.likes });
        onError(error);
      });
    },
    [qc, swipe, deck, onError],
  );
}
