export type Gender = "woman" | "man" | "non_binary" | "other";

export const GENDERS: readonly { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "non_binary", label: "Non-binaire" },
  { value: "other", label: "Autre" }
];

export type AuthUser = {
  id: string;
  email: string;
  role: "user" | "admin";
  hasProfile: boolean;
  photoCount: number;
  profileComplete: boolean;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type InterestRef = { slug: string; label: string };

export type PhotoRef = { id: string; position: number; url: string };

export type Photo = PhotoRef & { width: number; height: number; createdAt: string };

/** What other members see — no email, birth date or coordinates. */
export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: InterestRef[];
  photos: PhotoRef[];
};

export type OwnProfile = {
  id: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  interests: InterestRef[];
  photos: PhotoRef[];
  complete: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type ProfileInput = {
  firstName: string;
  birthDate?: string;
  gender: Gender;
  bio: string;
  city: string;
  interests?: string[];
  discoverable?: boolean;
  showDistance?: boolean;
};

export type SwipeAction = "like" | "pass";

export type MatchSummary = {
  id: string;
  createdAt: string;
  conversationId: string;
  hasMessages: boolean;
  user: PublicProfile;
};

export type SwipeResponse = {
  action: SwipeAction;
  matched: boolean;
  alreadySwiped: boolean;
  match?: MatchSummary;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type ConversationSummary = {
  id: string;
  matchId: string;
  user: PublicProfile;
  lastMessage: Message;
  unreadCount: number;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export const REPORT_REASONS = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
] as const;

export type ReportReason = (typeof REPORT_REASONS)[number]["value"];

/** Server push events (see services/api/internal/platform/realtime). */
export type RealtimeEvent =
  | { type: "message"; data: Message }
  | { type: "read"; data: { conversationId: string; readerId: string; readAt: string } }
  | { type: "match"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "unmatch"; data: { matchId: string } }
  | { type: "notification"; data: AppNotification };
