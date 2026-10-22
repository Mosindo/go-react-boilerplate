import { useInfiniteQuery } from "@tanstack/react-query";
import { fetchConversations } from "../api/chat";
import { fetchNotifications } from "../api/notifications";

/** Paginated conversation list. Shared by the Messages tab and the tab bar badge. */
export function useConversationsQuery() {
  return useInfiniteQuery({
    queryKey: ["conversations"],
    queryFn: ({ pageParam }) => fetchConversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 15_000
  });
}

/** Paginated notifications. Shared by the Notifications tab and the tab bar badge. */
export function useNotificationsQuery() {
  return useInfiniteQuery({
    queryKey: ["notifications"],
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 15_000
  });
}
