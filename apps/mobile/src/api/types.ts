// Types mirror docs/API.md exactly. Timestamps are RFC 3339 strings, ids are UUID strings.

export type Gender = "man" | "woman" | "non_binary";
export const GENDERS: readonly Gender[] = ["man", "woman", "non_binary"];

export type ReportReason = "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";
export const REPORT_REASONS: readonly ReportReason[] = [
  "spam",
  "fake_profile",
  "harassment",
  "inappropriate_content",
  "underage",
  "other"
];

export type ErrorCode =
  | "invalid_request"
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "rate_limited"
  | "profile_incomplete"
  | "underage"
  | "blocked"
  | "payload_too_large"
  | "unsupported_media"
  | "internal"
  | "network"
  | "timeout"
  | "unknown";

export type Page<T> = { items: T[]; nextCursor: string | null };

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };
export type AuthSession = AuthResponse;

export type Interest = { id: number; slug: string; label: string };
export type Photo = { id: string; position: number; url: string };

export type Profile = {
  userId: string;
  firstName: string;
  age: number;
  birthDate: string;
  gender: Gender;
  bio: string;
  locationLabel: string;
  hasLocation: boolean;
  showDistance: boolean;
  isDiscoverable: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Me = {
  id: string;
  email: string;
  createdAt: string;
  profile: Profile | null;
  preferences: Preferences;
  profileComplete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  interestIds: number[];
  showDistance?: boolean;
  isDiscoverable?: boolean;
};

export type LocationInput = { latitude: number; longitude: number; label?: string };

export type Candidate = {
  userId: string;
  firstName: string;
  age: number;
  bio: string;
  distanceKm: number | null;
  locationLabel: string;
  interests: Interest[];
  sharedInterestCount: number;
  photos: Photo[];
};

export type SwipeAction = "like" | "pass";

export type MatchUser = { userId: string; firstName: string; age: number; photo: Photo | null };
export type LastMessage = { body: string; senderId: string; createdAt: string };
export type MatchSummary = {
  matchId: string;
  conversationId: string;
  createdAt: string;
  user: MatchUser;
  lastMessage: LastMessage | null;
  unreadCount: number;
};
export type SwipeResult = { matched: boolean; match?: MatchSummary };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

/** A message as held in the client cache, possibly not yet acknowledged by the server. */
export type ChatMessage = Message & { status?: "sending" | "failed" };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type NotificationType = "match" | "message";
export type AppNotification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  readAt: string | null;
  createdAt: string;
};
export type NotificationPage = Page<AppNotification> & { unreadCount: number };

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "messages.read"; data: { conversationId: string; readerId: string } }
  | { type: "match.new"; data: MatchSummary }
  | { type: "match.removed"; data: { matchId: string; conversationId: string } }
  | { type: "notification.new"; data: AppNotification };
