export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "undecided";

export type AuthUser = {
  id: string;
  email: string;
  isDemo: boolean;
  profileComplete: boolean;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { id: number; slug: string; label: string; category: string };

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type PhotoRef = { id: string; url: string; position: number };

export type MyProfile = {
  userId: string;
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  hasLocation: boolean;
  discoverable: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences | null;
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  interestIds: number[];
};

export type CardInterest = { id: number; label: string; shared: boolean };

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  distanceKm: number | null;
  interests: CardInterest[];
  photos: PhotoRef[];
  isDemo: boolean;
};

export type CardSummary = {
  userId: string;
  firstName: string;
  age: number;
  photoUrl: string | null;
  isDemo: boolean;
};

export type SwipeAction = "like" | "pass";

export type SwipeResponse = {
  matched: boolean;
  matchId: string | null;
  conversationId: string | null;
  matchedUser: CardSummary | null;
};

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
  user: CardSummary;
  createdAt: string;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { conversationId?: string; matchId?: string };
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message.created"; data: Message }
  | { type: "conversation.read"; data: { conversationId: string; readerId: string; lastReadAt: string } }
  | { type: "match.created"; data: { matchId: string; conversationId: string; user: CardSummary } }
  | { type: "match.removed"; data: { matchId?: string; conversationId?: string } }
  | { type: "notification.created"; data: AppNotification };
