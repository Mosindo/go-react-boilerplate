export type Gender = "man" | "woman" | "nonbinary";

export type Interest = { slug: string; label: string };

export type AccountUser = {
  id: string;
  email: string;
  profileComplete: boolean;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AccountUser;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interests: string[];
};

export type Profile = Omit<ProfileInput, "interests"> & {
  userId: string;
  age: number;
  interests: Interest[];
  photoIds: string[];
  updatedAt: string;
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
  photoIds: string[];
};

export type Card = PublicProfile & { sharedInterests: number };

export type Settings = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
  discoverable: boolean;
  showDistance: boolean;
};

export type Photo = {
  id: string;
  position: number;
  width: number;
  height: number;
  createdAt: string;
};

export type SwipeResult = {
  action: "like" | "pass";
  matched: boolean;
  matchId?: string;
  conversationId?: string;
};

export type MatchSummary = {
  id: string;
  conversationId: string;
  createdAt: string;
  userId: string;
  firstName: string;
  age: number;
  photoId: string | null;
  hasMessages: boolean;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  userId: string;
  firstName: string;
  photoId: string | null;
  lastMessage: Message | null;
  unreadCount: number;
  updatedAt: string;
  otherLastReadAt: string;
};

export type MessagePage = {
  items: Message[];
  nextCursor: string;
  otherLastReadAt: string;
};

export type Paged<T> = { items: T[]; nextCursor: string };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actorId: string | null;
  actorName: string;
  refId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type NotificationPage = Paged<AppNotification> & { unreadCount: number };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "spam" | "harassment" | "fake" | "inappropriate" | "underage" | "other";

export type MatchEvent = {
  matchId: string;
  conversationId: string;
  userId: string;
  firstName: string;
  photoId?: string;
};
