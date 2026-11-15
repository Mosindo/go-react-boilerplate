export const queryKeys = {
  me: ["me"] as const,
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  likes: ["likes"] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversations", id] as const,
  messages: (id: string) => ["messages", id] as const,
  notifications: ["notifications"] as const,
  badges: ["badges"] as const,
  blocks: ["blocks"] as const,
  profile: (userId: string) => ["profiles", userId] as const
};
