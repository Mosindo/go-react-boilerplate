import { useMutation, useQueryClient } from "@tanstack/react-query";
import { matchingApi } from "../../api/matching";
import { queryKeys } from "../../api/queryKeys";
import type { SwipeAction } from "../../api/types";

/** Profiles swiped during this session, wherever the swipe happened. */
export const swipedIds = new Set<string>();

export function useSwipe() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (v: { userId: string; action: SwipeAction }) => matchingApi.swipe(v.userId, v.action),
    onMutate: (v) => {
      swipedIds.add(v.userId);
    },
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
      void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
      if (result.matched) void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  });
}
