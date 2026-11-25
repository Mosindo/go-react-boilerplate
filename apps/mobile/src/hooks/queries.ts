import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi } from "../api/chat";
import { notificationsApi } from "../api/notifications";
import { profileApi } from "../api/profile";
import { safetyApi } from "../api/safety";
import type { MeProfile, MessagesPage } from "../api/types";
import { insertMessage } from "./cache";
import { queryKeys } from "./queryClient";
import { useRealtimeConnected } from "./useRealtime";

// When the WebSocket is down, lists fall back to light polling.
const POLL_WHEN_OFFLINE_MS = 20_000;

export function useMyProfile(enabled = true) {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.me, enabled });
}

/** Writes a fresh MeProfile (returned by every profile mutation) to the cache. */
export function useSetMyProfile() {
  const qc = useQueryClient();
  return (profile: MeProfile) => {
    qc.setQueryData(queryKeys.myProfile, profile);
    void qc.invalidateQueries({ queryKey: queryKeys.discover });
  };
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => profileApi.publicProfile(userId) });
}

export function useConversations() {
  const connected = useRealtimeConnected();
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => (last.hasMore ? pages.reduce((n, p) => n + p.conversations.length, 0) : undefined),
    refetchInterval: connected ? false : POLL_WHEN_OFFLINE_MS
  });
}

export function useConversation(conversationId: string) {
  return useQuery({ queryKey: queryKeys.conversation(conversationId), queryFn: () => chatApi.conversation(conversationId) });
}

export function useMessages(conversationId: string) {
  const connected = useRealtimeConnected();
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: connected ? false : 8_000
  });
}

export function useChatUnread(enabled = true) {
  const connected = useRealtimeConnected();
  return useQuery({
    queryKey: queryKeys.chatUnread,
    queryFn: chatApi.unreadCount,
    enabled,
    refetchInterval: connected ? false : POLL_WHEN_OFFLINE_MS
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotificationsUnread(enabled = true) {
  const connected = useRealtimeConnected();
  return useQuery({
    queryKey: queryKeys.notificationsUnread,
    queryFn: notificationsApi.unreadCount,
    enabled,
    refetchInterval: connected ? false : POLL_WHEN_OFFLINE_MS
  });
}

export function useBlocks() {
  return useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
}

export function useSendMessage(conversationId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onSuccess: (message) => {
      qc.setQueryData<InfiniteData<MessagesPage, string | null>>(queryKeys.messages(conversationId), (d) => insertMessage(d, message));
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
    }
  });
}
