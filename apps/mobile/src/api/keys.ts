export const keys = {
  me: ["me"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  profile: (id: string) => ["profile", id] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  unread: ["unread"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  notificationsUnread: ["notifications", "unread"] as const,
  blocked: ["blocked"] as const
};
