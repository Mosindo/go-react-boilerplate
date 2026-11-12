export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "unsure";
export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type AuthUser = {
  id: string;
  email: string;
  organizationId: string;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { id: number; slug: string; label: string; category: string };

export type Photo = { id: string; url: string; width: number; height: number; position: number };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Privacy = { discoverable: boolean; showAge: boolean; showDistance: boolean };

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  relationshipGoal: RelationshipGoal;
  city: string;
  hasLocation: boolean;
  locationUpdatedAt?: string;
  privacy: Privacy;
  preferences: Preferences;
  interests: Interest[];
  photos: Photo[];
  isComplete: boolean;
  missing: ("preferences" | "photos")[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age?: number;
  gender: Gender;
  bio: string;
  occupation: string;
  relationshipGoal: RelationshipGoal;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  occupation: string;
  relationshipGoal: RelationshipGoal;
  city: string;
};

export type Match = {
  id: string;
  user: PublicProfile;
  createdAt: string;
  lastMessageAt?: string;
};

export type SwipeResult = { decision: "like" | "pass"; matched: boolean; match?: Match };

export type Message = {
  id: string;
  conversationId: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
};

export type MessagesPage = { messages: Message[]; otherLastReadAt?: string; hasMore: boolean };

export type ChatSummary = {
  matchId: string;
  conversationId: string;
  user: PublicProfile;
  lastMessage?: { id: string; senderUserId: string; content: string; createdAt: string };
  unreadCount: number;
  matchedAt: string;
};

export type AppNotification = {
  id: string;
  userId: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; photoUrl?: string; blockedAt: string };
