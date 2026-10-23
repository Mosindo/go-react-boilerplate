import { useMutation, useQueryClient } from "@tanstack/react-query";
import { swipe } from "../api";
import type { Card, MatchesPage, SwipeAction } from "../api/types";
import { errorMessage } from "../i18n/errorMessage";
import { t } from "../i18n/fr";
import { applyMatchNew, type Paged } from "../realtime/cache";
import { queryKeys } from "../session/queryKeys";
import { useToast } from "../ui";
import { discoverStore } from "./discoverStore";

/** Records a like/pass. The card leaves the deck immediately and comes back if the call fails. */
export function useSwipe() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: ({ card, action }: { card: Card; action: SwipeAction }) =>
      swipe(card.userId, action),
    onMutate: ({ card }) => discoverStore.markSwiped(card.userId),
    onSuccess: (result, { card }) => {
      if (result.matched && result.match) {
        const match = result.match;
        queryClient.setQueryData<Paged<MatchesPage>>(queryKeys.matches, (data) =>
          applyMatchNew(data, match)
        );
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        discoverStore.celebrate({ match, card });
      }
    },
    onError: (error, { card }) => {
      discoverStore.unmarkSwiped(card.userId);
      toast.error(`${t.discover.swipeFailed} ${errorMessage(error)}`);
    }
  });
  return {
    decide: (card: Card, action: SwipeAction) => mutation.mutate({ card, action }),
    isPending: mutation.isPending
  };
}
