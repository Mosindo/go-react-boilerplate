export const queryKeys = {
  profile: ["profile", "me"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  publicProfile: (userId: string) => ["profile", "public", userId] as const,
  conversations: ["conversations"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
