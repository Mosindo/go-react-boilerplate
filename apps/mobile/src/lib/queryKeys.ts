export const queryKeys = {
  profile: ["profile", "own"] as const,
  interests: ["interests"] as const,
  feed: ["discover", "feed"] as const,
  received: ["likes", "received"] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  notificationBadge: ["notifications", "badge"] as const,
  blocked: ["blocked"] as const,
  card: (id: string) => ["card", id] as const
};
