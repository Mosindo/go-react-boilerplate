export const keys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  preferences: ["preferences"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  publicProfile: (userId: string) => ["public-profile", userId] as const
};
