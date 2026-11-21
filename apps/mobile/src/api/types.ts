export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "undecided";

export type AuthUser = {
  id: string;
  email: string;
  organizationId: string;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { id: number; slug: string; label: string };

export type Photo = {
  id: string;
  position: number;
  url: string;
  thumbUrl: string;
  width: number;
  height: number;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type OwnProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
};

export type MeProfile = {
  userId: string;
  email: string;
  profile: OwnProfile | null;
  preferences: Preferences | null;
  photos: Photo[];
  interests: Interest[];
  isComplete: boolean;
  missing: string[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  distanceKm: number | null;
  isDemo: boolean;
  photos: Photo[];
  interests: Interest[];
};

export type FeedItem = ProfileCard & {
  likedYou: boolean;
  sharedInterests: number;
};

export type UserSummary = {
  id: string;
  firstName: string;
  age: number;
  photoUrl: string | null;
  isDemo: boolean;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  match?: { id: string; user: UserSummary; createdAt: string };
};

export type ChatSummary = {
  matchId: string;
  user: UserSummary;
  matchedAt: string;
  lastMessage?: { id: string; senderUserId: string; content: string; createdAt: string };
  unreadCount: number;
};

export type ChatMessage = {
  id: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
  isRead: boolean;
};

export type MessagesPage = {
  messages: ChatMessage[];
  hasMore: boolean;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = { user: UserSummary; blockedAt: string };

export type UpsertProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  city: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
};
