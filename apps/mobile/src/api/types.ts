// Shared API types. Mirrors docs/API.md exactly; keep in sync with the contract.

export type Gender = "man" | "woman" | "nonbinary";
export type Photo = { id: string; url: string; position: number };
export type Interest = { slug: string; label: string };
export type UserSummary = { userId: string; firstName: string; age: number; photo: Photo | null };
export type User = {
  id: string;
  email: string;
  createdAt: string;
  hasProfile: boolean;
  profileComplete: boolean;
};
export type SelfProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
  photos: Photo[];
  complete: boolean;
};
export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
  discoverable: boolean;
  showDistance: boolean;
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
  sharedInterests: string[];
  photos: Photo[];
};
export type Match = { id: string; user: UserSummary; conversationId: string; createdAt: string };
export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  content: string;
  createdAt: string;
};
export type Conversation = {
  id: string;
  matchId: string;
  user: UserSummary;
  lastMessage: Message | null;
  unreadCount: number;
  peerLastReadAt: string | null;
  updatedAt: string;
};
export type NotificationType = "match" | "message";
export type Notification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: { conversationId?: string; userId?: string; matchId?: string };
  isRead: boolean;
  createdAt: string;
};

export type ListResponse<T> = { items: T[] };
export type AuthSession = { accessToken: string; refreshToken: string; user: User };

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
};

export type ApiErrorCode =
  | "validation_error"
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "already_swiped"
  | "profile_incomplete"
  | "underage"
  | "blocked"
  | "rate_limited"
  | "internal"
  | "network_error"
  | "timeout";
