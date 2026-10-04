/** Types mirror docs/API.md. Keep in sync with the backend contract. */

export type Gender = "man" | "woman" | "nonbinary";
export const GENDERS: readonly Gender[] = ["man", "woman", "nonbinary"];

export type Photo = { id: string; position: number; url: string; thumbUrl: string };
export type Interest = { slug: string; label: string };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Me = { id: string; email: string; profileCompleted: boolean; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: Me };

export type OwnProfile = {
  userId: string;
  displayName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  interests: Interest[];
  photos: Photo[];
  showDistance: boolean;
  discoverable: boolean;
  hasLocation: boolean;
  preferences: Preferences;
  completed: boolean;
  createdAt: string;
  updatedAt: string;
};

export type PublicProfile = {
  userId: string;
  displayName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  interests: Interest[];
  photos: Photo[];
  distanceKm: number | null;
  commonInterests: number;
};

export type ProfileCard = { userId: string; displayName: string; age: number; photo: Photo | null };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

/** A message in the local cache; `status` only exists for optimistic (unsent/failed) entries. */
export type ChatMessage = Message & { status?: "sending" | "failed" };

export type MatchSummary = {
  id: string;
  conversationId: string;
  matchedAt: string;
  user: ProfileCard;
  lastMessage: Message | null;
  unreadCount: number;
  updatedAt: string;
};

export type NotificationType = "match" | "system";
export type AppNotification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  readAt: string | null;
  createdAt: string;
};

export type Page<K extends string, T> = { nextCursor: string | null } & { [P in K]: T[] };

export type SwipeAction = "like" | "pass";
export type SwipeResult = { action: SwipeAction; matched: boolean; match: MatchSummary | null };

export type BlockedUser = { userId: string; displayName: string; blockedAt: string };

export const REPORT_REASONS = ["spam", "fake", "harassment", "inappropriate", "underage", "other"] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

export type ProfileInput = {
  displayName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
  showDistance: boolean;
  discoverable: boolean;
};

export type ApiErrorCode =
  | "validation_failed"
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "profile_incomplete"
  | "rate_limited"
  | "payload_too_large"
  | "internal"
  | "network"
  | "timeout"
  | "unknown";

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readAt: string } }
  | { type: "match.new"; data: MatchSummary }
  | { type: "match.removed"; data: { matchId: string; conversationId: string } }
  | { type: "notification.new"; data: AppNotification };
