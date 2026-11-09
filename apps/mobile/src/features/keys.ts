export const keys = {
  profile: ["profile", "me"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  /** Prefix of every conversation query: invalidate this to refresh both the list and the unread badge. */
  conversations: ["conversations"] as const,
  conversationList: ["conversations", "list"] as const,
  conversationUnread: ["conversations", "unread"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  publicProfile: (userId: string) => ["profile", userId] as const
};
