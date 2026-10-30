export type Gender = "woman" | "man" | "non_binary";

export type Interest = { slug: string; label: string };
export type PhotoRef = { id: string; url: string };

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type OwnProfile = {
  userId: string;
  displayName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  isDiscoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences;
  isComplete: boolean;
  missing: string[];
};

export type PublicProfile = {
  userId: string;
  displayName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type SaveProfileInput = {
  displayName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude?: number;
  longitude?: number;
  interests: string[];
  isDiscoverable?: boolean;
  showDistance?: boolean;
};

export type AuthUser = { id: string; email: string; createdAt: string };

export type Session = { accessToken: string; refreshToken: string; user: AuthUser };

export type Photo = { id: string; url: string; position: number; width: number; height: number };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readByOther: boolean;
};

export type Conversation = {
  id: string;
  matchId: string;
  user: { userId: string; displayName: string; photoUrl?: string };
  lastMessage?: Message;
  unreadCount: number;
  lastActivityAt: string;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  duplicate: boolean;
  match?: { matchId: string; conversationId: string; user: PublicProfile };
};

export type AppNotification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: { conversationId?: string; userId?: string; matchId?: string };
  isRead: boolean;
  createdAt: string;
};

export type ReportReason = "spam" | "fake_profile" | "inappropriate_content" | "harassment" | "underage" | "other";

export type BlockedUser = { userId: string; displayName: string; blockedAt: string };

export type RealtimeEvent = {
  seq: number;
  type: "message" | "read" | "match" | "notification" | "unmatch";
  conversationId?: string;
};
