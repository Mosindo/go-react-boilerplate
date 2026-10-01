export type Gender = "man" | "woman" | "non_binary";

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type Interest = { id: number; slug: string; label: string };

export type PhotoRef = { id: string; position: number; url: string; thumbUrl: string };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type SelfProfile = {
  userId: string;
  exists: boolean;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender | "";
  bio: string;
  city: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences;
  complete: boolean;
  missing: string[];
};

/** Public view of another member: never includes coordinates, email or birth date. */
export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Account = { id: string; email: string; createdAt: string };

export type AuthSession = { accessToken: string; refreshToken: string; user: Account };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Conversation = {
  id: string;
  matchId: string;
  user: Card;
  lastMessage: Message | null;
  unreadCount: number;
  lastMessageAt: string;
};

export type Match = { id: string; conversationId: string; createdAt: string; user: Card };

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  matchId?: string;
  conversationId?: string;
  match?: Card;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  refId: string;
  actorId: string | null;
  actorName: string;
  actorThumbUrl: string | null;
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readerId: string } }
  | { type: "match.new"; data: Match }
  | { type: "match.removed"; data: { matchId?: string; userId?: string } }
  | { type: "notification.new"; data: { id: string; type: string; refId: string } };
