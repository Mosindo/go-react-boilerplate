export type Gender = "woman" | "man" | "nonbinary";

export type User = { id: string; email: string; createdAt: string };

export type AuthResponse = { accessToken: string; refreshToken: string; user: User };

export type Photo = { id: string; url: string; position: number };

export type Interest = { slug: string; label: string };

export type Preferences = { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number };

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  isVisible: boolean;
  interests: Interest[];
  photos: Photo[];
  preferences: Preferences | null;
  isComplete: boolean;
  missing: string[];
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

export type UpsertProfileInput = {
  firstName: string;
  birthDate?: string;
  gender: Gender;
  bio?: string;
  city?: string;
  interests?: string[];
  isVisible?: boolean;
};

export type SwipeAction = "like" | "pass";

export type SwipeResponse = {
  result: "liked" | "passed" | "matched";
  alreadyProcessed: boolean;
  match?: { matchId: string; conversationId: string; profile?: PublicProfile };
};

export type Message = { id: string; conversationId: string; senderId: string; body: string; createdAt: string };

export type Conversation = {
  id: string;
  matchId: string;
  with: { userId: string; firstName: string; photoUrl: string | null };
  lastMessage: Message | null;
  unread: number;
  createdAt: string;
};

export type ConversationsPage = { conversations: Conversation[]; nextCursor?: string };

export type MessagesPage = { messages: Message[]; nextBefore?: string; peerLastReadId: string };

export type AppNotification = {
  id: string;
  type: "match" | "like" | "new_message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = { notifications: AppNotification[]; nextCursor?: string; unread: number };

export type BlockedUser = { userId: string; firstName: string; photoUrl: string | null; blockedAt: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "pong" }
  | { type: "message.new"; data: Message }
  | { type: "conversation.read"; data: { conversationId: string; readerId: string; lastReadId: string } }
  | { type: "conversation.removed"; data: { conversationId: string } }
  | { type: "match.new"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "notification.new"; data: AppNotification };
