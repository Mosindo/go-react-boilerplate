export type Gender = "man" | "woman" | "non_binary";
export type SwipeAction = "like" | "pass";
export type ReportReason = "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type Interest = { id: number; slug: string; label: string };

export type PhotoRef = { id: string; position: number; width: number; height: number; version: string };

export type User = { id: string; email: string; profileComplete: boolean; createdAt: string };

export type AuthResponse = { accessToken: string; refreshToken: string; user: User };

export type SelfProfile = {
  userId: string;
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  isDiscoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interestIds: number[];
  isDiscoverable?: boolean;
  showDistance?: boolean;
};

export type Preferences = { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number };

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Brief = { userId: string; firstName: string; age: number; photo?: PhotoRef };

export type Match = {
  id: string;
  conversationId: string;
  user: Brief;
  createdAt: string;
  hasMessages: boolean;
};

export type SwipeResult = { action: SwipeAction; matched: boolean; alreadySwiped: boolean; match?: Match };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
  readAt?: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  peer: Brief;
  lastMessage?: Message;
  unreadCount: number;
  lastMessageAt: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actor?: { userId: string; firstName: string; photo?: PhotoRef };
  conversationId?: string;
  createdAt: string;
  readAt?: string;
};

export type Page<T, K extends string> = { [P in K]: T[] } & { nextCursor?: string };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "match.new"; data: Match }
  | { type: "match.removed"; data: { matchId: string; conversationId: string } }
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readAt: string; readerId: string } }
  | { type: "notification.new"; data: { type: string; conversationId?: string } };
