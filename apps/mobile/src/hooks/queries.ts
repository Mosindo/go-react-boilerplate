import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, notificationsApi, profileApi } from "../api/endpoints";
import type { ConversationsPage, MessagesPage, NotificationsPage } from "../api/types";
import { insertMessage } from "./chatCache";
import { queryKeys } from "../queryClient";
import { useRealtimeStatus } from "../realtime/RealtimeProvider";

// --- Profile ----------------------------------------------------------------

export function useMyProfile(enabled = true) {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.mine, enabled });
}

export function useInterests() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: async () => (await profileApi.interests()).interests,
    staleTime: Infinity
  });
}

// --- Conversations ----------------------------------------------------------

export function useConversations() {
  const realtime = useRealtimeStatus();
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: ConversationsPage) => last.nextCursor ?? undefined,
    // Realtime keeps the list fresh; poll slowly only when it is down.
    refetchInterval: realtime === "open" ? false : 15_000
  });
}

export function useMessages(conversationId: string) {
  const realtime = useRealtimeStatus();
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: MessagesPage) => last.nextCursor ?? undefined,
    refetchInterval: realtime === "open" ? false : 5_000
  });
}

export function useSendMessage(conversationId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      insertMessage(client, message);
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  });
}

// --- Notifications ----------------------------------------------------------

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last: NotificationsPage) => last.nextCursor ?? undefined
  });
}

export function useUnreadNotifications(enabled = true) {
  const realtime = useRealtimeStatus();
  return useQuery({
    queryKey: queryKeys.unreadNotifications,
    queryFn: async () => (await notificationsApi.unreadCount()).unreadCount,
    enabled,
    refetchInterval: realtime === "open" ? false : 30_000
  });
}
