export const queryKeys = {
  me: ["me"] as const,
  interests: ["interests"] as const,
  profile: ["profile"] as const,
  preferences: ["preferences"] as const,
  photos: ["photos"] as const,
  discover: ["discover"] as const,
  card: (userId: string) => ["card", userId] as const,
  matches: ["matches"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
