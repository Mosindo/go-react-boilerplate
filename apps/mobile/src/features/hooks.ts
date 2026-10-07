import { useQuery } from "@tanstack/react-query";
import { fetchConversations } from "../api/chat";
import { fetchMatches } from "../api/discover";
import { fetchNotifications } from "../api/notifications";
import { queryKeys } from "./queryKeys";
import { newMatchesWithoutMessages, totalUnread } from "../realtime/events";

export function useConversationsQuery() {
  return useQuery({ queryKey: queryKeys.conversations, queryFn: fetchConversations, staleTime: 15_000 });
}

export function useMatchesQuery() {
  return useQuery({ queryKey: queryKeys.matches, queryFn: fetchMatches, staleTime: 15_000 });
}

export function useNotificationsQuery() {
  return useQuery({ queryKey: queryKeys.notifications, queryFn: fetchNotifications, staleTime: 15_000 });
}

export { newMatchesWithoutMessages, totalUnread };
