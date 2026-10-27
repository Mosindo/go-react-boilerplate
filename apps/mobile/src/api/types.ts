export type Gender = "man" | "woman" | "non_binary";

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Photo = { id: string; url: string; position: number; width: number; height: number };

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  interests: string[];
  photos: Photo[];
  preferences: Preferences;
  complete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude?: number;
  longitude?: number;
  interests: string[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: string[];
  photos: Photo[];
};

export type Interest = { id: number; slug: string; label: string };

export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; matchId?: string; conversationId?: string };

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: PublicProfile;
};
export type MatchPage = { matches: Match[]; nextCursor?: string };

export type Message = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type MessagePage = { messages: Message[]; nextCursor?: string; otherLastReadId: number };

export type Peer = { userId: string; firstName: string; photoId?: string };
export type Conversation = {
  id: string;
  matchId: string;
  peer: Peer;
  lastMessage: Message | null;
  unreadCount: number;
  updatedAt: string;
};
export type ConversationPage = { conversations: Conversation[]; nextCursor?: string };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actor?: Peer;
  refId?: string;
  readAt: string | null;
  createdAt: string;
};
export type NotificationPage = {
  notifications: AppNotification[];
  unreadCount: number;
  nextCursor?: string;
};

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message"; payload: Message }
  | { type: "read"; payload: { conversationId: string; userId: string; upToMessageId: number } }
  | { type: "match"; payload: { matchId: string; conversationId: string; userId: string } };
