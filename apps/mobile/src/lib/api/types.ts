export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "unsure";

export type Photo = { id: string; url: string };
export type Interest = { id: number; slug: string; label: string };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type ProfileField = "firstName" | "birthdate" | "gender" | "interestedIn" | "photos";

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate: string | null;
  age: number | null;
  gender: Gender | null;
  bio: string;
  jobTitle: string;
  relationshipGoal: RelationshipGoal | null;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  photos: Photo[];
  interests: Interest[];
  preferences: Preferences;
  completeness: { complete: boolean; missing: ProfileField[] };
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  relationshipGoal: RelationshipGoal | null;
  city: string;
  distanceKm: number | null;
  photos: Photo[];
  interests: Interest[];
  sharedInterests: number;
};

export type ProfileSummary = {
  userId: string;
  firstName: string;
  age: number | null;
  photo: Photo | null;
};

export type AuthUser = { id: string; email: string; role: "member" | "moderator"; createdAt: string };

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: ProfileSummary;
};

export type SwipeResult = { matched: boolean; match?: Match };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  matchedAt: string;
  user: ProfileSummary;
  lastMessage: Message | null;
  unreadCount: number;
};

export type ConversationsPage = { conversations: Conversation[]; nextCursor: string | null };
export type MessagesPage = { messages: Message[]; otherLastReadAt: string | null; nextCursor: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
  readAt?: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
  nextOffset?: number;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = { user: ProfileSummary; blockedAt: string };

export type ModerationReport = {
  id: string;
  reason: ReportReason;
  details: string;
  status: "open" | "reviewed" | "dismissed";
  createdAt: string;
  reviewedAt?: string;
  reportedUser: ProfileSummary | null;
  reportedProfile?: PublicProfile;
  openReportsOnUser: number;
  reportedUserSuspended: boolean;
};
