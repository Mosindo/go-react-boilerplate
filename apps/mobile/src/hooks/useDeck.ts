import { useCallback, useEffect, useReducer } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../api/client";
import { getDiscoverBatch, swipe } from "../api/discover";
import { queryKeys } from "../api/queryKeys";
import type { MatchSummary, SwipeAction } from "../api/types";
import { addMatch, type MatchPages } from "../domain/realtimeCache";
import { deckReducer, initialDeck, shouldPrefetch } from "../domain/swipeDeck";
import { showToast } from "../shared/feedback";

const BATCH_SIZE = 10;

/** True when the target can no longer be swiped (deleted, blocked, hidden); retrying would never succeed. */
function isUnavailable(error: unknown): boolean {
  return error instanceof ApiError && (error.status === 404 || error.status === 403 || error.status === 409);
}

export function useDeck(enabled: boolean, onMatch: (match: MatchSummary) => void) {
  const [state, dispatch] = useReducer(deckReducer, initialDeck);
  const queryClient = useQueryClient();

  const fetchMore = useCallback(async () => {
    dispatch({ type: "fetchStarted" });
    try {
      dispatch({ type: "fetchSucceeded", items: await getDiscoverBatch(BATCH_SIZE) });
    } catch {
      dispatch({ type: "fetchFailed" });
    }
  }, []);

  useEffect(() => {
    if (enabled && shouldPrefetch(state)) {
      void fetchMore();
    }
  }, [enabled, fetchMore, state]);

  const decide = useCallback(
    async (action: SwipeAction) => {
      const top = state.cards[0];
      if (!top) {
        return;
      }
      dispatch({ type: "swiped", userId: top.userId });
      try {
        const result = await swipe(top.userId, action);
        dispatch({ type: "swipeConfirmed", userId: top.userId });
        if (result.matched && result.match) {
          const match = result.match;
          queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => addMatch(data, match));
          onMatch(match);
        }
      } catch (error) {
        if (isUnavailable(error)) {
          dispatch({ type: "swipeDropped", userId: top.userId });
          showToast("This profile is no longer available.", "info");
        } else {
          dispatch({ type: "swipeRolledBack", userId: top.userId });
          showToast(errorMessage(error, "Could not save your choice. Try again."), "error");
        }
      }
    },
    [onMatch, queryClient, state.cards]
  );

  const refresh = useCallback(() => dispatch({ type: "reset" }), []);

  /** Removes a card that was blocked, without a swipe request. */
  const discardTop = useCallback((userId: string) => {
    dispatch({ type: "swiped", userId });
    dispatch({ type: "swipeDropped", userId });
  }, []);

  return { state, decide, refresh, discardTop };
}
