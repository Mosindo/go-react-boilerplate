export type Gender = "man" | "woman" | "non_binary";

export const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Woman" },
  { value: "man", label: "Man" },
  { value: "non_binary", label: "Non-binary" }
];

export type User = { id: string; email: string; createdAt: string; profileComplete: boolean };
export type Interest = { slug: string; label: string };
export type Photo = { id: string; position: number; url: string };

export type Profile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  interests: Interest[];
  photos: Photo[];
  distanceKm: number | null;
  birthDate?: string;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
  discoverable: boolean;
  showDistance: boolean;
};

export type UserSummary = { userId: string; firstName: string; photoUrl: string | null };
export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type MatchItem = { id: string; user: UserSummary; conversationId: string; createdAt: string };

export type ConversationItem = {
  id: string;
  user: UserSummary;
  lastMessage: Message | null;
  unreadCount: number;
  updatedAt: string;
};

export type NotificationItem = {
  id: string;
  type: "match" | "message";
  actor: UserSummary | null;
  data: Record<string, unknown>;
  readAt: string | null;
  createdAt: string;
};

export type AuthSession = { accessToken: string; refreshToken: string; user: User };

export type ReportReason =
  "spam" | "fake_profile" | "inappropriate_content" | "harassment" | "underage" | "other";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "spam", label: "Spam" },
  { value: "fake_profile", label: "Fake profile" },
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "harassment", label: "Harassment" },
  { value: "underage", label: "Appears to be under 18" },
  { value: "other", label: "Something else" }
];

export type MessagesPage = { messages: Message[]; otherLastReadAt: string | null; nextCursor: string | null };
export type ConversationsPage = { conversations: ConversationItem[]; nextCursor: string | null };
export type MatchesPage = { matches: MatchItem[]; nextCursor: string | null };
export type NotificationsPage = {
  notifications: NotificationItem[];
  unreadCount: number;
  nextCursor: string | null;
};

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "pong" }
  | { type: "message.new"; message: Message }
  | { type: "message.read"; conversationId: string; userId: string; readAt: string }
  | { type: "match.new"; match: MatchItem }
  | { type: "match.removed"; matchId: string; conversationId: string }
  | { type: "notification.new"; notification: NotificationItem };
