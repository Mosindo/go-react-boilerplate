import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getMatches, unmatch } from "../api/matches";
import { queryKeys } from "../api/queryKeys";
import type { MatchSummary } from "../api/types";
import { removeMatch, totalUnread, type MatchPages } from "../domain/realtimeCache";

export function useMatches() {
  return useInfiniteQuery({
    queryKey: queryKeys.matches,
    queryFn: ({ pageParam }) => getMatches(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor ?? undefined
  });
}

export function flattenMatches(data: MatchPages | undefined): MatchSummary[] {
  return data ? data.pages.flatMap((page) => page.items) : [];
}

export function useUnreadMessageCount(): number {
  const { data } = useMatches();
  return totalUnread(data);
}

export function useUnmatch() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (matchId: string) => unmatch(matchId),
    onSuccess: (_result, matchId) => {
      queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => removeMatch(data, matchId));
    }
  });
}
