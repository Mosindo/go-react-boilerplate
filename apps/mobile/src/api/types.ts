export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "undecided";

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type Interest = { id: number; slug: string; label: string };
export type Photo = { id: string; url: string; width: number; height: number; position?: number };

export type OwnProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  hasLocation: boolean;
  isDiscoverable: boolean;
  showDistance: boolean;
  moderationStatus: "active" | "under_review" | "suspended";
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MissingStep = "profile" | "preferences" | "photos" | "location";

export type MeProfile = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  photos: Photo[];
  interests: Interest[];
  completion: { complete: boolean; missing: MissingStep[] };
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  interestIds: number[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  distanceKm: number | null;
  photos: Photo[];
  interests: Interest[];
};

export type Message = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Match = {
  id: string;
  createdAt: string;
  lastActivityAt: string;
  profile: ProfileCard;
  lastMessage: Message | null;
  unreadCount: number;
};

export type Page<T> = { items: T[]; nextCursor: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actorId: string | null;
  matchId: string | null;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "scam" | "underage" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; matchId: string | null };

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { matchId: string; readerId: string; readAt: string } }
  | { type: "match.new"; data: Match }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.new"; data: AppNotification };
