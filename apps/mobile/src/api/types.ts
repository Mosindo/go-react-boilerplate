export type Gender = "man" | "woman" | "nonbinary";

export type Interest = { id: number; slug: string; label: string };
export type Photo = { id: string; url: string; position: number };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

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
  showDistance: boolean;
  interests: Interest[];
  photos: Photo[];
  preferences: Preferences | null;
  onboardingComplete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interestIds: number[];
  isVisible?: boolean;
  showDistance?: boolean;
  latitude?: number;
  longitude?: number;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  matchId?: string;
  conversationId?: string;
  alreadySwiped: boolean;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Peer = { userId: string; firstName: string; photoUrl?: string };

export type Conversation = {
  id: string;
  matchId: string;
  peer: Peer;
  lastMessage: Message | null;
  unreadCount: number;
  createdAt: string;
  sortAt: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actorId: string;
  actorName: string;
  actorPhotoUrl?: string;
  matchId: string;
  conversationId?: string;
  readAt: string | null;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message"; data: Message }
  | { type: "read"; data: { conversationId: string } }
  | { type: "match"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "unmatch"; data?: unknown }
  | { type: "error"; data?: unknown };
