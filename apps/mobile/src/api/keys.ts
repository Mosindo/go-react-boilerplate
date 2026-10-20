export const queryKeys = {
  me: ["me"] as const,
  myProfile: ["profile", "me"] as const,
  interests: ["interests"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
