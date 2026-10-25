// Mirrors docs/API.md. JSON is camelCase, timestamps are RFC 3339 UTC.

export type Gender = "man" | "woman" | "nonbinary";

export type Photo = { id: string; position: number; isPrimary: boolean; url: string; thumbUrl: string };

export type Interest = { slug: string; label: string };

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
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
  sharedInterests: string[];
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type ProfileSummary = { userId: string; firstName: string; thumbUrl: string | null };

export type Page<T> = { items: T[]; nextCursor: string | null };

export type Me = { id: string; email: string; createdAt: string; profileComplete: boolean };

export type AuthResponse = { accessToken: string; refreshToken: string; user: Me };

export type ProfileState = { profile: MyProfile | null; preferences: Preferences | null; complete: boolean };

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  interests: string[];
  discoverable: boolean;
  showDistance: boolean;
};

export type LocationInput = { latitude: number; longitude: number; city: string };

export type SwipeAction = "like" | "pass";
export type SwipeResult = "liked" | "passed" | "matched" | "already_swiped";

export type MatchItem = { matchId: string; conversationId: string; createdAt: string; profile: PublicProfile };

export type SwipeResponse = { result: SwipeResult; match: MatchItem | null };

export type ConversationItem = {
  id: string;
  matchId: string;
  otherUser: ProfileSummary;
  lastMessage: Message;
  unreadCount: number;
  updatedAt: string;
};

export type BlockedUser = { userId: string; firstName: string; thumbUrl: string | null; createdAt: string };

export const REPORT_REASONS = [
  "spam",
  "fake_profile",
  "harassment",
  "inappropriate_content",
  "underage",
  "other"
] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

export type NotificationType = "match" | "message";

export type AppNotification = {
  id: string;
  type: NotificationType;
  actor: ProfileSummary | null;
  conversationId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type NotificationsPage = Page<AppNotification> & { unreadCount: number };

export type ReadReceipt = { conversationId: string; readerId: string; readAt: string };

export type RealtimeEvent =
  | { type: "message"; data: Message }
  | { type: "read"; data: ReadReceipt }
  | { type: "match"; data: MatchItem }
  | { type: "conversation_removed"; data: { conversationId: string } }
  | { type: "notification"; data: AppNotification };
