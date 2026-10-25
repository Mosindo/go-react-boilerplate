export const queryKeys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  publicProfile: (userId: string) => ["publicProfile", userId] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  unreadCount: ["unreadCount"] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
