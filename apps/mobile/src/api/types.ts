export type Gender = "man" | "woman" | "non_binary";

export type Interest = { slug: string; label: string };
export type PhotoRef = { id: string; url: string; position: number };

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender?: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type MyProfile = ProfileCard & {
  birthDate: string;
  discoverable: boolean;
  showDistance: boolean;
  hasLocation: boolean;
  complete: boolean;
  missing: ("gender" | "interestedIn" | "location" | "photos")[];
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthSession = { accessToken: string; refreshToken: string; user: AuthUser };

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  hasMessages: boolean;
  user: ProfileCard;
};
export type SwipeAction = "like" | "pass";
export type SwipeResult = { action: SwipeAction; matched: boolean; match?: Match };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type Conversation = {
  id: string;
  peer: ProfileCard;
  lastMessage?: Message;
  unreadCount: number;
  peerLastReadAt: string;
};
export type MessagesPage = { messages: Message[]; nextBefore?: string; peerLastReadAt: string };
export type ConversationsPage = { conversations: Conversation[]; nextBefore?: string };
export type MatchesPage = { matches: Match[]; nextBefore?: string };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actor?: { userId: string; firstName: string; photoUrl?: string };
  refId?: string;
  readAt?: string;
  createdAt: string;
};
export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
  nextBefore?: string;
};

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";
export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message.new"; data: Message }
  | { type: "match.new"; data: Match }
  | { type: "notification.new"; data: { notification: AppNotification; unreadCount: number } }
  | { type: "conversation.read"; data: { conversationId: string; userId: string; readAt: string } };
