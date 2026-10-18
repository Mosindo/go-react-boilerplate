export type Gender = "man" | "woman" | "nonbinary";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type Photo = {
  id: string;
  position: number;
  url: string;
  thumbUrl: string;
  width: number;
  height: number;
};

export type Interest = {
  slug: string;
  label: string;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number | null;
};

export type MyProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  isVisible: boolean;
  showDistance: boolean;
  interests: Interest[];
  preferences: Preferences;
  photos: Photo[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interests: string[];
  preferences: Preferences;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type MatchInfo = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: PublicProfile;
};

export type SwipeResult = {
  matched: boolean;
  match?: MatchInfo;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
};

export type ConversationSummary = {
  id: string;
  matchId: string;
  user: { userId: string; firstName: string; thumbUrl: string | null };
  lastMessage: Message | null;
  unreadCount: number;
  createdAt: string;
  updatedAt: string;
};

export type ConversationsPage = {
  conversations: ConversationSummary[];
  unreadTotal: number;
  nextCursor?: string;
};

export type MessagesPage = {
  messages: Message[];
  nextCursor?: string;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
  nextCursor?: string;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readAt: string } }
  | { type: "match.new"; data: { matchId: string; conversationId: string; userId: string; firstName: string } }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.new"; data: Record<string, string> };
