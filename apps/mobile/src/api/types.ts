export type Gender = "woman" | "man" | "nonbinary";
export type LookingFor = "relationship" | "casual" | "friendship" | "not_sure";
export type SwipeAction = "like" | "pass";
export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = {
  slug: string;
  label: string;
  category: string;
};

export type Photo = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type Profile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  updatedAt: string;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  profile: Profile | null;
  preferences: Preferences | null;
  photos: Photo[];
  complete: boolean;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  city: string;
  distanceKm: number | null;
  photos: Photo[];
  interests: Interest[];
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  lookingFor: LookingFor;
  interests: string[];
};

export type Match = {
  id: string;
  user: PublicProfile;
  createdAt: string;
};

export type SwipeResult = {
  matched: boolean;
  match?: Match;
};

export type Message = {
  id: string;
  matchId: string;
  senderId: string;
  body: string;
  createdAt: string;
};

export type Conversation = {
  matchId: string;
  user: PublicProfile;
  createdAt: string;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match" | "message" | "system";
  title: string;
  body: string;
  data: { matchId?: string; userId?: string } | null;
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};
