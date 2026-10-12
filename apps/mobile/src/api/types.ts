export type Gender = "woman" | "man" | "non_binary";

export type Interest = { slug: string; label: string };
export type Photo = { id: string; url: string; thumbUrl: string };

/** Public view of another member. Never contains exact location, birth date or e-mail. */
export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type OwnProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  city: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type OnboardingStep = "profile" | "photos" | "preferences";

export type MyProfile = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  interests: Interest[];
  photos: Photo[];
  onboardingComplete: boolean;
  missing: OnboardingStep[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  occupation: string;
  city: string;
  interests: string[];
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  matched: boolean;
  matchId: string | null;
  alreadyProcessed: boolean;
};

export type MessagePreview = { id: string; senderId: string; body: string; createdAt: string };

export type MatchSummary = {
  id: string;
  createdAt: string;
  lastActivityAt: string;
  profile: Card;
  lastMessage: MessagePreview | null;
  unreadCount: number;
};

export type ChatMessage = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type MessagePage = {
  items: ChatMessage[];
  nextBefore: string;
  otherLastReadAt: string | null;
};

export type NotificationActor = { userId: string; firstName: string; thumbUrl: string };

export type AppNotification = {
  id: string;
  kind: "match" | "message";
  actor: NotificationActor | null;
  matchId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type NotificationPage = {
  items: AppNotification[];
  unreadCount: number;
  nextBefore: string;
};

export type BlockedUser = { userId: string; firstName: string; thumbUrl: string; blockedAt: string };

export type ReportReason = "spam" | "fake" | "inappropriate" | "harassment" | "underage" | "other";

export type AuthUser = { id: string; email: string; createdAt: string };

export type ServerEvent =
  | { type: "ready" }
  | { type: "match.new"; data: { matchId: string } }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "message.new"; data: ChatMessage }
  | { type: "message.read"; data: { matchId: string; readAt: string } }
  | { type: "notification.new" };
