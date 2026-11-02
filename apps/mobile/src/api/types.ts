export type Gender = "woman" | "man" | "non_binary";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Une femme",
  man: "Un homme",
  non_binary: "Non-binaire"
};

export const GENDER_TARGET_LABELS: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  non_binary: "Des personnes non-binaires"
};

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = { id: number; slug: string; label: string };
export type PhotoRef = { id: string; position: number; url: string };

export type Profile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number | null;
  longitude: number | null;
  showDistance: boolean;
  discoverable: boolean;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type ProfileBundle = {
  profile: Profile | null;
  preferences: Preferences | null;
  complete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number | null;
  longitude: number | null;
  showDistance: boolean;
  discoverable: boolean;
  interestIds: number[];
};

export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  sharedInterests: number;
  photos: PhotoRef[];
};

export type SwipeResult = { action: "like" | "pass"; matched: boolean; matchId?: string };

export type Partner = { userId: string; firstName: string; photoUrl: string | null };

export type Conversation = {
  matchId: string;
  partner: Partner;
  createdAt: string;
  lastMessage: { body: string; senderId: string; createdAt: string } | null;
  unreadCount: number;
  activityAt: string;
};

export type Message = {
  id: number;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type NotificationItem = {
  id: string;
  type: "match" | "message";
  data: { matchId?: string; userId?: string; firstName?: string; photoUrl?: string; preview?: string };
  read: boolean;
  createdAt: string;
};

export type Page<T> = { items: T[]; nextCursor: string | number | null };

export type ReportReason = "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];
