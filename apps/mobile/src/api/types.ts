export type Gender = 'man' | 'woman' | 'nonbinary';
export const GENDERS: readonly Gender[] = ['man', 'woman', 'nonbinary'];

export type Photo = { id: string; position: number; url: string };
export type Interest = { slug: string; label: string };

export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type ProfileSummary = {
  id: string;
  firstName: string;
  age: number;
  mainPhotoUrl: string | null;
};

export type OnboardingState = {
  profile: boolean;
  photos: boolean;
  preferences: boolean;
  complete: boolean;
};

export type AuthUser = { id: string; email: string; createdAt: string };
export type Me = AuthUser & { onboarding: OnboardingState };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type MyProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interests: Interest[];
  photos: Photo[];
  discoverable: boolean;
  showDistance: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio?: string;
  city?: string;
  latitude: number;
  longitude: number;
  interests: string[];
  discoverable?: boolean;
  showDistance?: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type SwipeAction = 'like' | 'pass';
export type SwipeResult = {
  matched: boolean;
  alreadySwiped: boolean;
  match: null | { id: string; conversationId: string };
};

export type MatchItem = {
  id: string;
  createdAt: string;
  conversationId: string;
  user: ProfileSummary;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
  /** Client-only delivery state for optimistic messages. */
  localState?: 'sending' | 'failed';
};

export type LastMessage = { id: string; body: string; senderId: string; createdAt: string };

export type Conversation = {
  id: string;
  matchId: string;
  user: ProfileSummary;
  lastMessage: LastMessage | null;
  unreadCount: number;
  updatedAt: string;
};

export type MessagesPage = { messages: Message[]; hasMore: boolean };

export type BlockItem = { user: ProfileSummary; createdAt: string };

export const REPORT_REASONS = [
  'spam',
  'fake',
  'harassment',
  'inappropriate_photos',
  'underage',
  'other',
] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

export type NotificationType = 'match' | 'message';
export type AppNotification = {
  id: string;
  type: NotificationType | string;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  isRead: boolean;
  readAt: string | null;
  createdAt: string;
};
export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export type RealtimeEvent =
  | { type: 'message.new'; data: Message }
  | { type: 'message.read'; data: { conversationId: string; readerId: string; readAt: string } }
  | {
      type: 'match.new';
      data: { matchId: string; conversationId: string; user: ProfileSummary };
    }
  | { type: 'match.removed'; data: { matchId: string; conversationId: string } }
  | { type: 'notification.new'; data: AppNotification };
