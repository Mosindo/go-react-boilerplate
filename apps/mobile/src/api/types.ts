export type Gender = "man" | "woman" | "non_binary";

export type Photo = {
  id: string;
  position: number;
  url: string;
  thumbUrl: string;
  width: number;
  height: number;
};

export type Interest = { slug: string; label: string };

export type Message = {
  id: string;
  matchId: string;
  senderId: string;
  content: string;
  createdAt: string;
  readAt: string | null;
};

export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: (Interest & { shared: boolean })[];
  photos: Photo[];
};

export type MatchSummary = {
  id: string;
  createdAt: string;
  lastActivityAt: string;
  unreadCount: number;
  user: { userId: string; firstName: string; thumbUrl: string | null };
  lastMessage: { content: string; senderId: string; createdAt: string } | null;
};

export type NotificationItem = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { matchId?: string; userId?: string };
  readAt: string | null;
  createdAt: string;
};

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type OnboardingState = { hasProfile: boolean; photoCount: number; completed: boolean };
export type Me = { id: string; email: string; createdAt: string; onboarding: OnboardingState };

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude?: number | null;
  longitude?: number | null;
  interests: string[];
};

export type Profile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
  discoverable: boolean;
  showDistance: boolean;
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = {
  action: SwipeAction;
  alreadyProcessed: boolean;
  matched: boolean;
  match: MatchSummary | null;
};

export type MatchesPage = { matches: MatchSummary[]; nextCursor: string | null };
export type MessagesPage = { messages: Message[]; nextCursor: string | null };
export type NotificationsPage = {
  notifications: NotificationItem[];
  unreadCount: number;
  nextCursor: string | null;
};

export type BlockedUser = { userId: string; firstName: string; createdAt: string };

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type ReportInput = {
  userId: string;
  reason: ReportReason;
  details?: string;
  matchId?: string;
  block?: boolean;
};

export type ServerEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { matchId: string; readerId: string; readAt: string } }
  | { type: "match.new"; data: MatchSummary }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.new"; data: NotificationItem };
