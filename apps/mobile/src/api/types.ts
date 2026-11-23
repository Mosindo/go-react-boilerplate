export type Gender = "woman" | "man" | "nonbinary";

export type Interest = { slug: string; label: string };

export type Photo = { id: string; position: number; url: string };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Settings = {
  isDiscoverable: boolean;
  hideDistance: boolean;
};

export type MissingField = "firstName" | "birthdate" | "gender" | "interestedIn" | "photo" | "location";

export type MyProfile = {
  userId: string;
  firstName: string;
  birthdate: string | null;
  age: number | null;
  gender: Gender | "";
  bio: string;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
  photos: Photo[];
  preferences: Preferences;
  settings: Settings;
  missing: MissingField[];
  isComplete: boolean;
  maxPhotos: number;
  maxInterests: number;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type LastMessage = { content: string; senderUserId: string; createdAt: string };

export type MatchSummary = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: { userId: string; firstName: string; age: number; photo: Photo | null };
  lastMessage: LastMessage | null;
  unreadCount: number;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = { action: SwipeAction; matched: boolean; match?: MatchSummary };

export type Message = {
  id: string;
  conversationId: string;
  senderUserId: string;
  content: string;
  createdAt: string;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  otherUserId: string;
  otherLastReadAt: string | null;
};

export type ReadReceipt = { conversationId: string; userId: string; lastReadAt: string };

export type AppNotification = {
  id: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number; hasMore: boolean };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "scam" | "underage" | "other";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const audienceLabels: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil",
  inappropriate_content: "Contenu inapproprié",
  harassment: "Harcèlement ou propos offensants",
  scam: "Arnaque ou demande d'argent",
  underage: "Personne mineure",
  other: "Autre raison"
};
