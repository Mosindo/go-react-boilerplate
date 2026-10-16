export type Gender = "woman" | "man" | "nonbinary";
export type LookingFor = "relationship" | "casual" | "friendship" | "open";
export type SwipeAction = "like" | "pass";

export type PhotoRef = { id: string; url: string };
export type Interest = { slug: string; label: string };

/** The only shape in which other people appear: no email, birth date or coordinates. */
export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  lookingFor: LookingFor;
  city: string;
  distanceKm: number | null;
  photos: PhotoRef[];
  interests: Interest[];
};

export type OwnProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  lookingFor: LookingFor;
  city: string;
  latitude: number;
  longitude: number;
  discoverable: boolean;
  showDistance: boolean;
  interests: string[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MeProfile = {
  profile: OwnProfile | null;
  preferences: Preferences | null;
  photos: PhotoRef[];
  isComplete: boolean;
  missing: string[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  occupation: string;
  lookingFor: LookingFor;
  city: string;
  latitude: number;
  longitude: number;
  interests: string[];
};

export type SwipeResult = { matched: boolean; matchId?: string };

export type Match = { id: string; createdAt: string; profile: PublicProfile };

export type Message = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Conversation = {
  matchId: string;
  createdAt: string;
  activityAt: string;
  profile: PublicProfile;
  lastMessage: Message | null;
  unreadCount: number;
};

export type MessagesPage = { messages: Message[]; hasMore: boolean };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { matchId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};

export type BlockedUser = { blockedAt: string; profile: PublicProfile };

export type ReportReason = "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { matchId: string; readerId: string } }
  | { type: "match.new"; data: { matchId: string; userId: string } }
  | { type: "notification.new"; data: AppNotification };
