export type Gender = "man" | "woman" | "nonbinary";

export type Interest = { slug: string; label: string };

export type Photo = { id: string; url: string; position: number };

export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  interests: Interest[];
  photos: Photo[];
  distanceKm?: number;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
};

export type OwnProfile = PublicProfile & {
  birthDate: string;
  isVisible: boolean;
  showDistance: boolean;
  hasLocation: boolean;
  discoverable: boolean;
  preferences: Preferences;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
  latitude?: number;
  longitude?: number;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  matched: boolean;
  matchId?: string;
  profile?: PublicProfile;
};

export type ChatMessage = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Conversation = {
  matchId: string;
  user: { id: string; firstName: string; photoUrl?: string };
  lastMessage: { body: string; senderId: string; createdAt: string } | null;
  unreadCount: number;
  matchedAt: string;
  updatedAt: string;
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type BlockedUser = { id: string; firstName: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";
