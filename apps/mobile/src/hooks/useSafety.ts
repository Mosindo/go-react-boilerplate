import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as safetyApi from "../api/safety";
import { queryKeys } from "../api/queryKeys";
import type { ReportInput } from "../api/safety";
import { bumpDiscoverEpoch, pushDeckIntent } from "../store/deckIntent";

export function useBlocks() {
  return useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.listBlocks });
}

/** After blocking, the person disappears from matches, conversations and the deck. */
function useAfterBlock() {
  const client = useQueryClient();
  return (userId: string) => {
    pushDeckIntent({ userId, kind: "remove" });
    void client.invalidateQueries({ queryKey: queryKeys.blocks });
    void client.invalidateQueries({ queryKey: queryKeys.matchesAll });
    void client.invalidateQueries({ queryKey: queryKeys.conversationsAll });
    client.removeQueries({ queryKey: queryKeys.publicProfile(userId) });
  };
}

export function useBlockUser() {
  const afterBlock = useAfterBlock();
  return useMutation({
    mutationFn: (userId: string) => safetyApi.blockUser(userId),
    onSuccess: (_v, userId) => afterBlock(userId)
  });
}

export function useUnblockUser() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => safetyApi.unblockUser(userId),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      bumpDiscoverEpoch();
    }
  });
}

export function useReportUser() {
  const afterBlock = useAfterBlock();
  return useMutation({
    mutationFn: (input: ReportInput) => safetyApi.reportUser(input),
    onSuccess: (_v, input) => {
      if (input.block) afterBlock(input.userId);
    }
  });
}
