export const queryKeys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  conversations: ["conversations"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  unread: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const,
  profileOf: (userId: string) => ["profileOf", userId] as const
};
