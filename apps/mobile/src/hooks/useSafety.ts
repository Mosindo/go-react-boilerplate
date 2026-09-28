import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { blockUser, getBlockedUsers, reportUser, unblockUser } from "../api/safety";
import { queryKeys } from "../api/queryKeys";
import type { ReportReason } from "../api/types";

export function useBlockedUsers() {
  return useQuery({ queryKey: queryKeys.blocked, queryFn: getBlockedUsers });
}

export function useBlock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => blockUser(userId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
    }
  });
}

export function useUnblock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    }
  });
}

export function useReport() {
  return useMutation({
    mutationFn: (input: { userId: string; reason: ReportReason; details?: string }) =>
      reportUser(input.userId, input.reason, input.details)
  });
}
