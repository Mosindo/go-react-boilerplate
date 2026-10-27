export const keys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const,
  userProfile: (userId: string) => ["user-profile", userId] as const
};
