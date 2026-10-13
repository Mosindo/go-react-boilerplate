export type Gender = "woman" | "man" | "nonbinary";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type Interest = { id: number; slug: string; label: string };
export type PhotoRef = { id: string; position: number };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
};

export type OwnProfile = {
  id: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string | null;
  latitude: number | null;
  longitude: number | null;
  isVisible: boolean;
  showDistance: boolean;
  interestIds: number[];
  photos: PhotoRef[];
  preferences: Preferences | null;
  isComplete: boolean;
};

export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string | null;
  distanceKm: number | null;
  interests: Interest[];
  photos: PhotoRef[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string | null;
  interestIds: number[];
  isVisible?: boolean;
  showDistance?: boolean;
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; match?: { id: string; conversationId: string } };

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  lastMessageAt: string | null;
  profile: PublicProfile;
};

export type ChatMessage = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  read: boolean;
};

export type Conversation = {
  id: string;
  matchId: string;
  profile: PublicProfile;
  lastMessage: { body: string; senderId: string; createdAt: string };
  unreadCount: number;
  updatedAt: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { conversationId?: string; matchId?: string; userId?: string; name?: string };
  isRead: boolean;
  createdAt: string;
};

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready" | "pong" | "notification" }
  | { type: "message"; data: ChatMessage }
  | { type: "match"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "unmatch"; data?: { matchId: string } }
  | { type: "read"; data: { conversationId: string; readAt: string } };
