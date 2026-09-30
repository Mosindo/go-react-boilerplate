import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi } from "../../lib/api/endpoints";
import { queryKeys } from "../../lib/queryClient";
import { useRealtime } from "../../lib/realtime/RealtimeProvider";

/** Conversations, refreshed by realtime events; polls only when offline. */
export function useConversations() {
  const { connected } = useRealtime();
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: connected ? false : 20_000
  });
}

export function useUnreadMessagesCount(): number {
  const { data } = useConversations();
  return data?.pages.reduce((sum, page) => sum + page.conversations.reduce((s, c) => s + c.unreadCount, 0), 0) ?? 0;
}
