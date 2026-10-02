export type Gender = "woman" | "man" | "nonbinary";

export interface User {
  id: string;
  email: string;
  createdAt: string;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  user: User;
}

export interface Interest {
  slug: string;
  label: string;
  shared?: boolean;
}

export interface Photo {
  id: string;
  url: string;
  position: number;
}

/** Public view of someone else's profile. No coordinates, no birth date. */
export interface Card {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
}

export interface OwnProfile {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  discoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: Photo[];
}

export interface Preferences {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
}

export interface ProfileStatus {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  complete: boolean;
  missing: string[];
}

export interface ProfileInput {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interests: string[];
  discoverable?: boolean;
  showDistance?: boolean;
}

export interface LastMessage {
  body: string;
  senderId: string;
  createdAt: string;
}

export interface MatchItem {
  id: string;
  createdAt: string;
  user: Card;
  lastMessage?: LastMessage;
  unreadCount: number;
}

export interface MatchPage {
  matches: MatchItem[];
  nextCursor?: string;
}

export interface SwipeResult {
  action: "like" | "pass";
  matched: boolean;
  matchId?: string;
  user?: Card;
}

export interface Message {
  id: number;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt?: string;
}

export interface MessagePage {
  messages: Message[];
  hasMore: boolean;
}

export interface NotificationItem {
  id: string;
  type: "match" | "message";
  matchId: string;
  actor?: Card;
  createdAt: string;
  readAt?: string;
}

export interface NotificationPage {
  notifications: NotificationItem[];
  unread: number;
  nextCursor?: string;
}

export interface Summary {
  unreadNotifications: number;
  unreadMessages: number;
}

export interface BlockedUser {
  id: string;
  firstName: string;
  blockedAt: string;
}

export type ReportReason = "spam" | "fake" | "inappropriate" | "harassment" | "underage" | "other";

export type RealtimeEvent =
  | { type: "message"; payload: Message }
  | { type: "match"; payload: { matchId: string } }
  | { type: "read"; payload: { matchId: string } }
  | { type: "unmatched"; payload: { matchId: string } };
