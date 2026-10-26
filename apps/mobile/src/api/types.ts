export type Gender = "man" | "woman" | "nonbinary";

export type Interest = { slug: string; label: string };
export type Photo = { id: string; url: string };

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type OwnProfile = PublicProfile & {
  birthDate: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
  preferences: Preferences | null;
  isComplete: boolean;
  missing: ("preferences" | "location" | "photos")[];
  createdAt: string;
};

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type Message = {
  id: number;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Conversation = {
  matchId: string;
  matchedAt: string;
  profile: PublicProfile;
  lastMessage: Message | null;
  unreadCount: number;
  activityAt: string;
};

export type MatchView = { id: string; createdAt: string; profile: PublicProfile };

export type SwipeResult = { matched: boolean; alreadySwiped: boolean; match?: MatchView };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  matchId: string | null;
  actor: { userId: string; firstName: string; photoUrl: string | null } | null;
  readAt: string | null;
  createdAt: string;
};

export type BlockedUser = { userId: string; firstName: string; photoUrl: string | null; blockedAt: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message"; data: Message }
  | { type: "read"; data: { matchId: string; readerId: string; readAt: string; count: number } }
  | { type: "match"; data: MatchView }
  | { type: "match_ended"; data: { matchId: string } };
