import type { Gender, LookingFor } from "../lib/labels";

export type Photo = { id: string; url: string; thumbUrl: string; position: number };
export type Interest = { slug: string; label: string };
export type UserSummary = { id: string; firstName: string; thumbUrl: string };

export type Me = {
  id: string;
  email: string;
  age: number;
  createdAt: string;
  hasProfile: boolean;
  photoCount: number;
};

export type AuthResponse = { accessToken: string; refreshToken: string; user: Me };

export type OwnProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number | null;
  longitude: number | null;
  occupation: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  interests: string[];
  photos: Photo[];
};

export type ProfileInput = {
  firstName: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude?: number;
  longitude?: number;
  clearLocation?: boolean;
  occupation: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  interests: string[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  occupation: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
  discoverable: boolean;
  showDistance: boolean;
};

export type Match = { id: string; conversationId: string; createdAt: string; user: UserSummary };
export type SwipeAction = "like" | "pass";
export type SwipeResult = { action: SwipeAction; matched: boolean; match?: Match };

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
  user: UserSummary;
  lastMessage: { body: string; senderId: string; createdAt: string } | null;
  unreadCount: number;
  updatedAt: string;
};

export type MessagePage = { items: Message[]; nextBefore: number | null; otherLastReadAt: string | null };
export type ConversationPage = { items: Conversation[]; nextBefore: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  conversationId: string | null;
  actor: UserSummary;
  readAt: string | null;
  createdAt: string;
};
export type NotificationPage = { items: AppNotification[]; unreadCount: number; nextBefore: string | null };

export type BlockedUser = { user: UserSummary; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready"; data: null }
  | { type: "message.new"; data: Message }
  | { type: "match.new"; data: Match }
  | { type: "conversation.read"; data: { conversationId: string; userId: string; readAt: string } }
  | { type: "conversation.removed"; data: { conversationId: string } }
  | { type: "notification.new"; data: { type: string } }
  | { type: "error"; data: string };
