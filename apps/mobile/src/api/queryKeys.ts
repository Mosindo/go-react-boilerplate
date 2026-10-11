export const queryKeys = {
  me: ['me'] as const,
  interests: ['interests'] as const,
  ownProfile: ['profile', 'own'] as const,
  preferences: ['profile', 'preferences'] as const,
  privacy: ['profile', 'privacy'] as const,
  profile: (userId: string) => ['profile', 'public', userId] as const,
  likes: ['likes'] as const,
  conversations: ['conversations'] as const,
  messages: (conversationId: string) => ['messages', conversationId] as const,
  notifications: ['notifications'] as const,
  blocks: ['blocks'] as const,
};
