export type Gender = "woman" | "man" | "nonbinary";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const GENDER_PLURAL_LABELS: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export type AuthUser = {
  id: string;
  email: string;
  organizationId: string;
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
};

export type Photo = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  userId: string;
  exists: boolean;
  isComplete: boolean;
  missing: string[];
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender | "";
  bio: string;
  city: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
  isDemo: boolean;
  photos: Photo[];
  interests: Interest[];
  preferences: Preferences;
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city?: string;
  distanceKm?: number;
  isDemo: boolean;
  photos: Photo[];
  interests: Interest[];
  sharedInterests: number;
};

export type SwipeAction = "like" | "pass";

export type LastMessage = {
  id: string;
  senderUserId: string;
  content: string;
  createdAt: string;
};

export type MatchSummary = {
  id: string;
  matchedAt: string;
  user: ProfileCard;
  lastMessage?: LastMessage;
  unreadCount: number;
};

export type SwipeResult = {
  action: SwipeAction;
  alreadySwiped: boolean;
  matched: boolean;
  match?: MatchSummary;
};

export type ChatMessage = {
  id: string;
  matchId: string;
  senderUserId: string;
  content: string;
  clientId?: string;
  createdAt: string;
  isRead: boolean;
  /** Client-only state for optimistic sends. */
  pending?: boolean;
  failed?: boolean;
};

export type MessagesPage = {
  messages: ChatMessage[];
  otherLastReadAt?: string;
  nextCursor?: string;
};

export type MatchesPage = {
  matches: MatchSummary[];
  nextCursor?: string;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
  readAt?: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  nextCursor?: string;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "harassment", label: "Harcèlement ou propos menaçants" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "fake_profile", label: "Faux profil ou usurpation" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre raison" }
];
