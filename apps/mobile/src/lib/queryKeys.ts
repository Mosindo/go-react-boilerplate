export const queryKeys = {
  overview: ["overview"] as const,
  interests: ["interests"] as const,
  feed: ["feed"] as const,
  profile: (id: string) => ["profile", id] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  unread: ["unread"] as const,
  blocks: ["blocks"] as const
};
