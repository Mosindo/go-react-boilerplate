import { useMutation, useQueryClient } from "@tanstack/react-query";
import * as Haptics from "expo-haptics";
import { Platform } from "react-native";
import { ApiError } from "../../api/client";
import { discoverApi } from "../../api/endpoints";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { keys } from "../../hooks/queries";
import { useToast } from "../../ui/Toast";
import { useMatchCelebration } from "./matchCelebration";
import { markSwiped, unmarkSwiped } from "./swipedMemory";

type Vars = { card: ProfileCard; action: SwipeAction };

/** Like or pass a profile. The card leaves the deck immediately and comes back if the server refuses. */
export function useSwipe() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const celebration = useMatchCelebration();

  return useMutation({
    mutationFn: ({ card, action }: Vars) => discoverApi.swipe(card.userId, action),
    onMutate: ({ card }) => {
      markSwiped(card.userId);
      queryClient.setQueryData<ProfileCard[]>(keys.discover, (old) =>
        old?.filter((c) => c.userId !== card.userId)
      );
      if (Platform.OS !== "web") void Haptics.selectionAsync();
    },
    onSuccess: (result) => {
      if (result.matched && result.match) {
        celebration.show(result.match);
        void queryClient.invalidateQueries({ queryKey: keys.matches });
        void queryClient.invalidateQueries({ queryKey: keys.conversations });
      }
    },
    onError: (error, { card }) => {
      // 409/404: the server already has an answer for this profile (or it is gone) — keep it out of the deck.
      const final = error instanceof ApiError && (error.status === 409 || error.status === 404);
      if (!final) {
        unmarkSwiped(card.userId);
        queryClient.setQueryData<ProfileCard[]>(keys.discover, (old) => [card, ...(old ?? [])]);
      }
      toast.show(error instanceof ApiError ? error.message : "Action impossible pour le moment.", "error");
    }
  });
}
