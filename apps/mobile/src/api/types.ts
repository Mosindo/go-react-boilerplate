export type Gender = "man" | "woman" | "nonbinary";

export type User = { id: string; email: string; createdAt: string };

export type AuthResponse = { accessToken: string; refreshToken: string; user: User };

export type Interest = { slug: string; label: string };
export type PhotoRef = { id: string; url: string; position: number };

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type Privacy = { showDistance: boolean; discoverable: boolean };

export type MyProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences | null;
  privacy: Privacy;
  complete: boolean;
  missing: string[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
};

export type Card = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: PhotoRef[];
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; matchId?: string; conversationId?: string };

export type LikeReceived = { likedAt: string; user: Card };

export type Message = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  user: Card;
  lastMessage: Message | null;
  unreadCount: number;
  activityAt: string;
};

export type MessagesPage = { messages: Message[]; otherLastReadMessageId: number };

export type AppNotification = {
  id: string;
  type: "match" | "system";
  matchId: string | null;
  actor: { id: string; firstName: string; photoUrl: string | null } | null;
  readAt: string | null;
  createdAt: string;
};

export type BlockedUser = { id: string; firstName: string; photoUrl: string | null; blockedAt: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";
