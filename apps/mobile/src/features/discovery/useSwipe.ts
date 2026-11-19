import { useMutation, useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { SwipeResponse } from "../../api/types";
import { useMatchCelebration } from "../../state/matchCelebration";

type Listener = (userId: string) => void;
const listeners = new Set<Listener>();

/** Lets the deck drop a card that was answered from another screen (profile page, likes list). */
export function onSwiped(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Like/pass mutation shared by the deck, the profile page and the likes list. */
export function useSwipe() {
  const client = useQueryClient();
  const { celebrate } = useMatchCelebration();
  return useMutation({
    mutationFn: ({ userId, decision }: { userId: string; decision: "like" | "pass" }): Promise<SwipeResponse> =>
      decision === "like" ? discoveryApi.like(userId) : discoveryApi.pass(userId),
    onSuccess: (res, { userId }) => {
      listeners.forEach((l) => l(userId));
      void client.invalidateQueries({ queryKey: queryKeys.likesReceived });
      if (res.matched && res.match) {
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
        celebrate(res.match);
      }
    }
  });
}
