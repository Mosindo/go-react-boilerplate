export const queryKeys = {
  me: ["me"] as const,
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  chats: ["chats"] as const,
  messages: (userId: string) => ["messages", userId] as const,
  profile: (userId: string) => ["profile", userId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
