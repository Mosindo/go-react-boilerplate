export type Gender = "woman" | "man" | "nonbinary";
export type LookingFor = "relationship" | "casual" | "friendship" | "unsure";

export type AuthUser = {
  id: string;
  email: string;
  hasProfile: boolean;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { id: number; slug: string; label: string };

export type Photo = { id: string; url: string; position: number; width: number; height: number };

export type MyProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  city: string;
  hasLocation: boolean;
  isDiscoverable: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type Completeness = {
  hasProfile: boolean;
  hasPreferences: boolean;
  hasPhotos: boolean;
  hasLocation: boolean;
  isComplete: boolean;
};

export type MeProfile = {
  profile: MyProfile | null;
  preferences: Preferences | null;
  completeness: Completeness;
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  city: string;
  interestIds: number[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  city: string;
  distanceKm: number | null;
  interests: { id: number; label: string; shared: boolean }[];
  photos: Photo[];
};

export type DiscoverResponse = { profiles: ProfileCard[]; reason?: string };

export type SwipeAction = "like" | "pass";

export type Match = {
  id: string;
  createdAt: string;
  lastMessageAt: string | null;
  user: ProfileCard;
  lastMessage: { body: string; senderId: string; createdAt: string } | null;
  unreadCount: number;
};

export type SwipeResponse = { action: SwipeAction; matched: boolean; match?: Match };

export type MatchesResponse = { matches: Match[]; hasMore: boolean };

export type Message = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  clientMessageId: string | null;
  createdAt: string;
  readAt: string | null;
  /** Client-only: optimistic message not yet acknowledged. */
  pending?: boolean;
  failed?: boolean;
};

export type MessagesPage = { messages: Message[]; nextBefore: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actorId: string | null;
  matchId: string | null;
  title: string;
  body: string;
  isRead: boolean;
  createdAt: string;
  updatedAt: string;
  readAt: string | null;
};

export type NotificationsResponse = { notifications: AppNotification[]; unreadCount: number; hasMore: boolean };

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };
