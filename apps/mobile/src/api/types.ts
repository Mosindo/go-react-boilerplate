export type Gender = "man" | "woman" | "nonbinary";

export const GENDER_LABELS: Record<Gender, string> = {
  man: "Homme",
  woman: "Femme",
  nonbinary: "Non-binaire"
};

export type Account = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: Account };

export type PhotoRef = { id: string; position: number };
export type Interest = { slug: string; label: string };

export type OwnProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  interests: string[];
  photos: PhotoRef[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
};

export type Privacy = { isDiscoverable: boolean; showDistance: boolean };

export type MissingStep = "profile" | "preferences" | "photos";

export type Overview = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  privacy: Privacy | null;
  onboardingComplete: boolean;
  missing: MissingStep[];
};

export type Candidate = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: string[];
  sharedInterests: number;
  photos: PhotoRef[];
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = {
  matched: boolean;
  alreadySwiped: boolean;
  matchId?: string;
  conversationId?: string;
};

export type Message = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  read: boolean;
};

export type Conversation = {
  id: string;
  matchId: string;
  peer: { id: string; firstName: string; mainPhotoId: string | null };
  lastMessage: { id: number; senderId: string; body: string; createdAt: string } | null;
  unreadCount: number;
  matchedAt: string;
  activityAt: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  conversationId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export type BlockedUser = { id: string; firstName: string; blockedAt: string };
