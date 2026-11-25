export type Gender = "woman" | "man" | "nonbinary";
export type LookingFor = "relationship" | "casual" | "friendship" | "unsure";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = {
  id: number;
  slug: string;
  label: string;
};

export type Photo = {
  id: string;
  position: number;
  width: number;
  height: number;
  url: string;
};

export type OwnProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string | null;
  city: string | null;
  heightCm: number | null;
  lookingFor: LookingFor | null;
  interests: Interest[];
  hasLocation: boolean;
  locationUpdatedAt: string | null;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Privacy = {
  discoverable: boolean;
  showDistance: boolean;
};

export type CompletenessStep = "profile" | "location" | "preferences" | "photos";

export type MeProfile = {
  profile: OwnProfile | null;
  preferences: Preferences;
  privacy: Privacy;
  photos: Photo[];
  completeness: {
    isComplete: boolean;
    missing: CompletenessStep[];
  };
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string | null;
  city: string | null;
  heightCm: number | null;
  lookingFor: LookingFor | null;
  interestIds: number[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string | null;
  city: string | null;
  heightCm: number | null;
  lookingFor: LookingFor | null;
  distanceKm: number | null;
  interests: Interest[];
  commonInterestIds: number[];
  photos: Photo[];
  isDemo: boolean;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  alreadySwiped: boolean;
  match: {
    matchId: string;
    conversationId: string;
    profile: PublicProfile;
  } | null;
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
  matchedAt: string;
  peer: {
    userId: string;
    firstName: string;
    photo: Photo | null;
  };
  lastMessage: Message | null;
  unreadCount: number;
  peerLastReadAt: string | null;
};

export type MessagesPage = {
  messages: Message[];
  nextCursor: string | null;
  peerLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match" | "message" | "system";
  title: string;
  body: string;
  data: Record<string, string>;
  readAt: string | null;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  nextCursor: string | null;
  unreadCount: number;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};
