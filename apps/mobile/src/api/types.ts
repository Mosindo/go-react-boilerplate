export type Gender = "woman" | "man" | "nonbinary";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire",
};

export type Interest = { slug: string; label: string };
export type PhotoRef = { id: string; position: number };
export type Photo = PhotoRef & { width: number; height: number };

export type Me = { id: string; email: string; profileComplete: boolean; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: Me };

export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number | null;
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
  discoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences | null;
  complete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio?: string;
  city?: string;
  interests?: string[];
  discoverable?: boolean;
  showDistance?: boolean;
};

export type MatchInfo = { id: string; conversationId: string; user: Card };
export type SwipeAction = "like" | "pass";
export type SwipeResult = { action: SwipeAction; matched: boolean; match?: MatchInfo };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
  readAt?: string | null;
};

export type Conversation = {
  id: string;
  matchId: string;
  user: Card;
  lastMessage?: Message;
  unreadCount: number;
  updatedAt: string;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason =
  "spam" | "fake" | "harassment" | "inappropriate_content" | "underage" | "other";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" },
];

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message.new"; data: Message }
  | { type: "messages.read"; data: { conversationId: string; readerId: string; readAt: string } }
  | { type: "match.new"; data: MatchInfo }
  | { type: "match.removed"; data: { conversationId: string } }
  | { type: "notification.new"; data: AppNotification };
