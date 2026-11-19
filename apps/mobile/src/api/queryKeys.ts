export const queryKeys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  likesReceived: ["likes-received"] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  notifications: ["notifications"] as const,
  unreadNotifications: ["notifications", "unread"] as const,
  publicProfile: (id: string) => ["public-profile", id] as const,
  blocks: ["blocks"] as const
};
