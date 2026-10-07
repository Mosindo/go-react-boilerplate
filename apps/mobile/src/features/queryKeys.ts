export const queryKeys = {
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  matches: ["matches"] as const,
  notifications: ["notifications"] as const,
  userProfile: (userId: string) => ["user-profile", userId] as const
};
