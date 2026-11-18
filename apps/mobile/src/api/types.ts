export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "unsure";

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

export type Interest = { slug: string; label: string; category: string };

export type Photo = { id: string; position: number; url: string; width: number; height: number };

export type ProfileView = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
  hasLocation: boolean;
  locationUpdatedAt: string | null;
  isDiscoverable: boolean;
  showDistance: boolean;
  underReview: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  profile: ProfileView | null;
  preferences: Preferences | null;
  photos: Photo[];
  interests: Interest[];
  isComplete: boolean;
  missing: ("profile" | "preferences" | "photos")[];
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
};

export type CardInterest = { slug: string; label: string; shared: boolean };

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
  distanceKm: number | null;
  sharedInterests: number;
  interests: CardInterest[];
  photos: { id: string; url: string; width: number; height: number }[];
  isMatch: boolean;
  isDemo: boolean;
};

export type UserSummary = {
  userId: string;
  firstName: string;
  age: number;
  photoUrl: string | null;
};

export type Match = {
  id: string;
  user: UserSummary;
  matchedAt: string;
  hasMessages: boolean;
};

export type SwipeDecision = "like" | "pass";

export type SwipeResult = {
  decision: SwipeDecision;
  matched: boolean;
  match?: Match;
};

export type Message = {
  id: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
};

export type ChatSummary = {
  user: UserSummary;
  lastMessage: Message;
  unreadCount: number;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match.created" | "message.received" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  photoUrl: string | null;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";
