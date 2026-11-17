export const queryKeys = {
  me: ["me"] as const,
  profile: ["profile", "me"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  publicProfile: (userId: string) => ["profile", userId] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  unreadConversations: ["conversations", "unread"] as const,
  notifications: ["notifications"] as const,
  unreadNotifications: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};
