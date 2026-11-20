import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { matchingApi } from "../../api/endpoints";
import type { SwipeAction, SwipeResponse } from "../../api/types";
import { errorMessage } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { showToast } from "../../ui";
import { matchCelebrations } from "./matchCelebrations";
import { swipeStore } from "./swipeStore";

type Vars = { userId: string; action: SwipeAction };

/** Optimistic like/pass shared by the deck, the profile page and the likes list. */
export function useSwipe(onMatch?: (result: SwipeResponse) => void) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, action }: Vars) => matchingApi.swipe(userId, action),
    onMutate: ({ userId, action }) => {
      swipeStore.add(userId);
      if (action === "like") {
        matchCelebrations.markLike(userId);
      }
    },
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: keys.likes });
      if (result.matched && result.matchId) {
        void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
        onMatch?.(result);
      }
    },
    onError: (error, { userId }) => {
      // Already answered or gone: keep it out of the deck. Otherwise give it back.
      if (error instanceof ApiError && (error.code === "already_swiped" || error.status === 404)) {
        return;
      }
      swipeStore.remove(userId);
      showToast(errorMessage(error), "error");
    }
  });
}
