// Mirrors the JSON contracts of the Go API (services/api/internal/features/*/model.go).

export type Gender = "woman" | "man" | "nonbinary";

export type AuthUser = {
  id: string;
  email: string;
  isDemo: boolean;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: AuthUser;
};

export type Interest = {
  id: number;
  slug: string;
  label: string;
  category: string;
};

export type Photo = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type PrivacySettings = {
  discoverable: boolean;
  showDistance: boolean;
};

export type OnboardingStep = "firstName" | "birthdate" | "gender" | "photos" | "location";

export type MyProfile = {
  userId: string;
  firstName: string;
  birthdate: string | null;
  age: number | null;
  gender: Gender | null;
  bio: string;
  jobTitle: string;
  city: string;
  hasLocation: boolean;
  settings: PrivacySettings;
  preferences: Preferences;
  interests: Interest[];
  photos: Photo[];
  isDemo: boolean;
  isComplete: boolean;
  missing: OnboardingStep[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
  isDemo: boolean;
};

export type DiscoverCard = PublicProfile & { sharedInterests: number };

export type Match = {
  id: string;
  conversationId: string;
  otherUserId: string;
  createdAt: string;
  profile: PublicProfile;
};

export type SwipeAction = "like" | "pass";

export type SwipeResponse = {
  action: SwipeAction;
  matched: boolean;
  match: Match | null;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
  clientMessageId?: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  matchedAt: string;
  profile: PublicProfile;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
};

export type Page<K extends string, T> = { [key in K]: T[] } & { nextCursor: string | null };

export type MessagesPage = {
  messages: Message[];
  nextCursor: string | null;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match.new" | "message.new" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  nextCursor: string | null;
  unreadCount: number;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "scam" | "underage" | "other";
