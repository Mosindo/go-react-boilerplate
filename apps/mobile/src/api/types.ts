// Wire types of the Lien API (see services/api). Dates are ISO strings.

export type Gender = "man" | "woman" | "non_binary";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = {
  id: number;
  slug: string;
  label: string;
  /** Only set on other people's profiles: also one of mine. */
  shared?: boolean;
};

export type Photo = {
  id: string;
  position: number;
  /** Server-relative, authenticated: /photos/<id>/content */
  url: string;
};

export type OwnProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  showAge: boolean;
  showDistance: boolean;
  discoverable: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type OnboardingStep = "profile" | "location" | "preferences" | "photos";

export type SelfState = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  interests: Interest[];
  photos: Photo[];
  onboarding: { complete: boolean; missing: OnboardingStep[] };
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age?: number;
  gender: Gender;
  bio: string;
  city?: string;
  /** Rounded up to 5 km steps. */
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  alreadyProcessed: boolean;
  matched: boolean;
  matchId?: string;
};

export type ChatMessage = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type MatchSummary = {
  id: string;
  user: { userId: string; firstName: string; primaryPhotoUrl?: string };
  createdAt: string;
  lastMessage?: ChatMessage;
  unreadCount: number;
};

export type MatchesPage = { matches: MatchSummary[]; unreadTotal: number };

export type MessagesPage = {
  messages: ChatMessage[];
  hasMore: boolean;
  peerLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "new_match" | "new_message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "spam" | "fake_profile" | "inappropriate_content" | "harassment" | "underage" | "other";

export type ServerEvent =
  | { type: "ready" }
  /** Synthetic: the socket reconnected, so anything may have been missed. */
  | { type: "resync" }
  | { type: "message.new"; data: ChatMessage }
  | { type: "message.read"; data: { matchId: string; readAt: string } }
  | { type: "match.new"; data: { matchId: string; userId: string } }
  | { type: "match.ended"; data: { matchId: string } }
  | { type: "notification.new"; data: AppNotification };
