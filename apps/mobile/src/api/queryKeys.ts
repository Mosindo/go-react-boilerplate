/** Single source of truth for React Query keys. */
export const queryKeys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  preferences: ["preferences"] as const,
  interests: ["interests"] as const,
  publicProfile: (userId: string) => ["publicProfile", userId] as const,
  matches: ["matches", "list"] as const,
  matchesAll: ["matches"] as const,
  conversations: ["conversations", "list"] as const,
  conversationsAll: ["conversations"] as const,
  conversationsUnread: ["conversations", "unread"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  messagesAll: ["messages"] as const,
  notifications: ["notifications", "list"] as const,
  notificationsAll: ["notifications"] as const,
  notificationsUnread: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};
