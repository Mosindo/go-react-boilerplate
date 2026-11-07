export type Gender = "woman" | "man" | "nonbinary";

export type Interest = { slug: string; label: string };
export type PhotoView = { id: string; url: string };

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  showDistance: boolean;
  discoverable: boolean;
  interests: Interest[];
  preferences: Preferences;
  photos: PhotoView[];
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
  interests: string[];
  preferences: Preferences;
};

export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  photos: PhotoView[];
  interests: Interest[];
  sharedInterests: number;
};

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type MatchView = { id: string; conversationId: string; createdAt: string; user: Card };
export type SwipeResponse = { action: "like" | "pass"; matched: boolean; match?: MatchView };

export type Message = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type Peer = { userId: string; firstName: string; photoUrl: string | null };
export type ConversationSummary = {
  id: string;
  matchId: string;
  peer: Peer;
  lastMessage: Message | null;
  unreadCount: number;
  createdAt: string;
  sortAt: string;
};
export type MessagesPage = {
  messages: Message[];
  nextBefore: number | null;
  peerReadAt: string | null;
  peer: Peer;
};

export type NotificationItem = {
  id: string;
  type: "match" | "like";
  actorId: string | null;
  actorName: string | null;
  photoUrl: string | null;
  refId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; photoUrl: string | null; blockedAt: string };
export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type RealtimeEvent =
  | { type: "ready"; data: null }
  | { type: "message"; data: Message }
  | { type: "read"; data: { conversationId: string; readAt: string } }
  | { type: "match"; data: MatchView }
  | { type: "unmatch"; data: { matchId: string } }
  | { type: "notification"; data: { kind: string } };
