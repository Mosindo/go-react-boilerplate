export const queryKeys = {
  me: ["me"] as const,
  interests: ["interests"] as const,
  matches: ["matches"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const,
  candidate: (userId: string) => ["candidate", userId] as const
};
