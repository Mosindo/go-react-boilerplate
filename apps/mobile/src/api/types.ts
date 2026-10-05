export type Gender = "woman" | "man" | "non_binary";

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { slug: string; label: string };

export type Photo = { id: string; url: string; position: number; width?: number; height?: number };

/** Public view of another user. Never carries coordinates or a birth date. */
export type ProfileCard = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  city?: string;
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
};

export type OwnProfile = {
  id: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  city: string;
  hasLocation: boolean;
  isDiscoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Completion = {
  profile: boolean;
  preferences: boolean;
  location: boolean;
  photos: boolean;
  complete: boolean;
};

export type MyProfile = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  completion: Completion;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  occupation: string;
  interests: string[];
};

export type Message = {
  id: string;
  conversationId: string;
  senderUserId: string;
  content: string;
  createdAt: string;
  readAt?: string;
};

export type MessagePreview = { content: string; senderUserId: string; createdAt: string };

export type MatchPreview = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: ProfileCard;
  lastMessage?: MessagePreview;
  unreadCount: number;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = { action: SwipeAction; matched: boolean; match?: MatchPreview };

export type ConversationPreview = {
  id: string;
  matchId: string;
  user: ProfileCard;
  lastMessage?: Message;
  unreadCount: number;
};

export type MessagesPage = { messages: Message[]; nextCursor?: string };

export type AppNotification = {
  id: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, unknown>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export type ReportReason = "spam" | "fake_profile" | "inappropriate_content" | "harassment" | "underage" | "scam" | "other";

export type BlockedUser = { user: ProfileCard; blockedAt: string };

/** Events pushed by the API over the WebSocket. */
export type RealtimeEvent =
  | { type: "ready" }
  | { type: "match.new"; data: MatchPreview }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "message.new"; data: Message }
  | { type: "messages.read"; data: { conversationId: string; readerUserId: string; readAt: string } }
  | { type: "notification.new"; data: AppNotification };
