export const qk = {
  me: ['me'] as const,
  myProfile: ['profile', 'me'] as const,
  preferences: ['preferences'] as const,
  myPhotos: ['photos', 'me'] as const,
  interests: ['interests'] as const,
  profile: (userId: string) => ['profile', userId] as const,
  likes: ['likes'] as const,
  conversations: ['conversations'] as const,
  messages: (conversationId: string) => ['messages', conversationId] as const,
  notifications: ['notifications'] as const,
  blocks: ['blocks'] as const,
  matches: ['matches'] as const,
};
