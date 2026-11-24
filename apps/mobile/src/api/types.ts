// DTOs returned by the Lueur API. Keep in sync with services/api/internal/features/*/model.go.

export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "unsure";
export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type Account = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  user: Account;
};

export type Photo = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type Interest = {
  id: number;
  slug: string;
  label: string;
  category: string;
};

export type CardInterest = Interest & { shared: boolean };

export type Profile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
  city: string;
  hasLocation: boolean;
  isDiscoverable: boolean;
  showDistance: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  profile: Profile | null;
  preferences: Preferences;
  interests: Interest[];
  photos: Photo[];
  isComplete: boolean;
  missing: ("profile" | "photo" | "location")[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
  city: string;
  distanceKm: number | null;
  interests: CardInterest[];
  photos: Photo[];
};

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: ProfileCard;
};

export type SwipeDecision = "like" | "pass";

export type SwipeResponse = {
  decision: SwipeDecision;
  matched: boolean;
  match?: Match;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  isRead: boolean;
};

export type Conversation = {
  id: string;
  matchId: string;
  matchedAt: string;
  lastMessage: Message | null;
  unreadCount: number;
  otherUser: {
    userId: string;
    firstName: string;
    age: number;
    photo: Photo | null;
  };
};

export type ConversationsPage = {
  conversations: Conversation[];
  nextCursor: string | null;
};

export type MessagesPage = {
  messages: Message[];
  nextCursor: string | null;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, unknown>;
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
