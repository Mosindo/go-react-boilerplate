import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { swipe } from "../api/discovery";
import type { PublicProfile, SwipeAction, SwipeResult } from "../api/types";
import { showError } from "../shared/feedback";
import { keys } from "./queries";

/** Ids whose swipe is not confirmed by the server yet — kept out of the deck while a refetch races. */
const inFlight = new Set<string>();
export const isSwipeInFlight = (userId: string) => inFlight.has(userId);

const LOW_WATER_MARK = 3;

/**
 * Records a decision. The card leaves the deck immediately (optimistic); on failure it is put back
 * at the top so nothing is silently lost. The deck is refilled once the server has the swipe.
 */
export function useSwipe() {
  const queryClient = useQueryClient();

  return useCallback(
    async (profile: PublicProfile, action: SwipeAction): Promise<SwipeResult | null> => {
      inFlight.add(profile.userId);
      queryClient.setQueryData<PublicProfile[]>(keys.discover, (deck) => (deck ?? []).filter((p) => p.userId !== profile.userId));
      try {
        const result = await swipe(profile.userId, action);
        if (result.matched) {
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
        }
        return result;
      } catch (err) {
        queryClient.setQueryData<PublicProfile[]>(keys.discover, (deck) => [
          profile,
          ...(deck ?? []).filter((p) => p.userId !== profile.userId)
        ]);
        showError(err instanceof Error ? err.message : "Action impossible pour le moment.");
        return null;
      } finally {
        inFlight.delete(profile.userId);
        const remaining = queryClient.getQueryData<PublicProfile[]>(keys.discover)?.length ?? 0;
        if (remaining <= LOW_WATER_MARK) void queryClient.invalidateQueries({ queryKey: keys.discover });
      }
    },
    [queryClient]
  );
}
