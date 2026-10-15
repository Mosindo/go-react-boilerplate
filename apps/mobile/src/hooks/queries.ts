import { useInfiniteQuery, useQuery, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { getMyProfile, getProfile, listBlocked, listConversations, listInterests, listMessages, listNotifications } from "../api/platform";
import type { Message, MessagesPage } from "../api/types";

export const queryKeys = {
  myProfile: ["profile", "me"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  interests: ["interests"] as const,
  chats: ["chats"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: getMyProfile, staleTime: 60_000 });
}

export function useProfile(userId: string) {
  return useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => getProfile(userId), retry: false });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });
}

export function useConversations() {
  return useQuery({ queryKey: queryKeys.chats, queryFn: listConversations, staleTime: 15_000 });
}

export function useNotifications() {
  return useQuery({ queryKey: queryKeys.notifications, queryFn: listNotifications, staleTime: 15_000 });
}

export function useBlocked() {
  return useQuery({ queryKey: queryKeys.blocked, queryFn: listBlocked });
}

/** Newest-first pages; `before` is the id of the oldest message already loaded. */
export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => listMessages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[last.messages.length - 1]?.id : undefined)
  });
}

type MessagesData = InfiniteData<MessagesPage, string | undefined>;

/** Adds a message to the cache of its conversation once (REST answer and WebSocket push can both deliver it). */
export function appendMessage(client: QueryClient, message: Message): void {
  client.setQueryData<MessagesData>(queryKeys.messages(message.conversationId), (old) => {
    if (!old || old.pages.length === 0) {
      return old;
    }
    if (old.pages.some((page) => page.messages.some((m) => m.id === message.id))) {
      return old;
    }
    const [first, ...rest] = old.pages;
    return { ...old, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
  });
}

export function setPeerLastRead(client: QueryClient, conversationId: string, readAt: string): void {
  client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (old) => {
    if (!old || old.pages.length === 0) {
      return old;
    }
    const [first, ...rest] = old.pages;
    return { ...old, pages: [{ ...first, peerLastReadAt: readAt }, ...rest] };
  });
}
