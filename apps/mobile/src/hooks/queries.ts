import { useQuery } from "@tanstack/react-query";
import { fetchConversations } from "../api/chat";
import { fetchNotifications } from "../api/notifications";

export const keys = {
  discover: ["discover"] as const,
  profile: ["profile"] as const,
  photos: ["photos"] as const,
  preferences: ["preferences"] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  publicProfile: (userId: string) => ["public-profile", userId] as const
};

export function useConversations() {
  return useQuery({ queryKey: keys.conversations, queryFn: fetchConversations, select: (page) => page.items });
}

export function useNotifications() {
  return useQuery({ queryKey: keys.notifications, queryFn: fetchNotifications });
}

/** Unread counters for tab and header badges. */
export function useUnreadCounts(): { messages: number; notifications: number } {
  const conversations = useConversations();
  const notifications = useNotifications();
  return {
    messages: (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0),
    notifications: notifications.data?.unreadCount ?? 0
  };
}
