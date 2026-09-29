import { useInfiniteQuery, type InfiniteData } from "@tanstack/react-query";
import {
  CONVERSATIONS_PAGE_SIZE,
  MESSAGES_PAGE_SIZE,
  listConversations,
  listMessages
} from "../api/chat";
import { NOTIFICATIONS_PAGE_SIZE, listNotifications } from "../api/notifications";
import type { ConversationsPage, NotificationsPage } from "../lib/dating/cache";
import type { MessagesPage } from "../lib/dating/messages";

export const CONVERSATIONS_KEY = ["conversations"] as const;
export const NOTIFICATIONS_KEY = ["notifications"] as const;
export const messagesKey = (conversationId: string) => ["messages", conversationId] as const;

export type ConversationsData = InfiniteData<ConversationsPage, string | undefined>;
export type NotificationsData = InfiniteData<NotificationsPage, string | undefined>;
export type MessagesData = InfiniteData<MessagesPage, string | undefined>;

/** Infinite list of conversations, keyset-paginated on `updatedAt`. */
export function useConversationsQuery(options?: {
  enabled?: boolean;
  refetchInterval?: number | false;
}) {
  return useInfiniteQuery({
    queryKey: CONVERSATIONS_KEY,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listConversations(pageParam),
    getNextPageParam: (last): string | undefined => {
      if (last.conversations.length < CONVERSATIONS_PAGE_SIZE) return undefined;
      return last.conversations[last.conversations.length - 1]?.updatedAt;
    },
    enabled: options?.enabled ?? true,
    refetchInterval: options?.refetchInterval ?? false
  });
}

/** Infinite list of notifications, keyset-paginated on `createdAt`. */
export function useNotificationsQuery(options?: {
  enabled?: boolean;
  refetchInterval?: number | false;
}) {
  return useInfiniteQuery({
    queryKey: NOTIFICATIONS_KEY,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    getNextPageParam: (last): string | undefined => {
      if (last.notifications.length < NOTIFICATIONS_PAGE_SIZE) return undefined;
      return last.notifications[last.notifications.length - 1]?.createdAt;
    },
    enabled: options?.enabled ?? true,
    refetchInterval: options?.refetchInterval ?? false
  });
}

/** Messages of one conversation, newest first, cursor = id of the oldest loaded message. */
export function useMessagesQuery(conversationId: string) {
  return useInfiniteQuery({
    queryKey: messagesKey(conversationId),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => listMessages(conversationId, pageParam, MESSAGES_PAGE_SIZE),
    getNextPageParam: (last): string | undefined => last.nextCursor ?? undefined
  });
}
