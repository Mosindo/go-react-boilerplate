import { useMutation, useQueryClient } from "@tanstack/react-query";
import { matchingApi } from "../api/endpoints";
import type { ProfileCard, SwipeDecision } from "../api/types";
import { showToast } from "../components/Toast";
import { queryKeys } from "../queryClient";
import { announceMatch } from "../realtime/matchAnnouncements";
import { errorCode, errorMessage } from "../utils/errors";
import { celebrateMatch } from "../components/discover/matchCelebration";

export function removeFromDeck(client: ReturnType<typeof useQueryClient>, userId: string) {
  client.setQueryData<ProfileCard[]>(queryKeys.discovery, (cards) => cards?.filter((c) => c.userId !== userId));
}

/** Records a like/pass. The card leaves the deck immediately (optimistic). */
export function useSwipe() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ card, decision }: { card: ProfileCard; decision: SwipeDecision }) => matchingApi.swipe(card.userId, decision),
    onMutate: ({ card }) => removeFromDeck(client, card.userId),
    onSuccess: (result) => {
      if (result.matched && result.match) {
        announceMatch(result.match.id);
        celebrateMatch(result.match);
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (error) => {
      const code = errorCode(error);
      if (code !== "already_swiped" && code !== "profile_unavailable") {
        showToast(errorMessage(error), "error");
      }
    }
  });
}
