export type Gender = "woman" | "man" | "nonbinary";

export type AuthUser = {
  id: string;
  email: string;
  isDemo: boolean;
  createdAt: string;
};

export type AuthSession = {
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
  url: string;
  width: number;
  height: number;
  position: number;
};

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate: string | null;
  age: number | null;
  gender: Gender | null;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  city: string;
  hasLocation: boolean;
  isPaused: boolean;
  hideDistance: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MissingField = "firstName" | "birthdate" | "gender" | "photo";

export type MeProfile = {
  profile: OwnProfile;
  preferences: Preferences;
  completeness: { isComplete: boolean; missing: MissingField[] };
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender | "";
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  city: string;
  distanceKm: number | null;
  isDemo: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type MatchSummary = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: ProfileCard;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  matched: boolean;
  match: MatchSummary | null;
};

export type ReceivedLike = {
  likedAt: string;
  user: ProfileCard;
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
  user: ProfileCard;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
};

export type Page<K extends string, T> = { [key in K]: T[] } & { nextCursor: string | null };

export type MessagesPage = {
  messages: Message[];
  nextCursor: string | null;
  otherLastReadAt: string | null;
};

export type NotificationType = "match" | "message" | "like";

export type AppNotification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: Record<string, string>;
  readAt: string | null;
  createdAt: string;
};

export type Badges = {
  unreadNotifications: number;
  unreadConversations: number;
  likesReceived: number;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";
