export type Gender = "woman" | "man" | "nonbinary";

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
  slug: string;
  label: string;
  category: string;
};

export type Photo = {
  id: string;
  url: string;
  width: number;
  height: number;
  position: number;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Completion = {
  profile: boolean;
  preferences: boolean;
  photos: boolean;
  complete: boolean;
};

export type MyProfile = {
  userId: string;
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender | "";
  bio: string;
  jobTitle: string;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: Photo[];
  preferences: Preferences | null;
  completion: Completion;
};

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
  isDemo: boolean;
};

export type UpdateProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  interests: string[];
};

export type DeckResponse = {
  profiles: PublicProfile[];
  reason?: "no_more_profiles" | "no_more_profiles_location_off";
};

export type SwipeAction = "like" | "pass";

export type Match = {
  id: string;
  conversationId: string;
  createdAt: string;
  hasMessages: boolean;
  profile: PublicProfile;
};

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  match: Match | null;
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
  profile: PublicProfile;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
  createdAt: string;
};

export type Page<T, K extends string> = { [key in K]: T[] } & { nextCursor: string | null };

export type MessagesPage = {
  messages: Message[];
  nextCursor: string | null;
  otherLastReadAt: string | null;
};

export type ReadReceipt = {
  conversationId: string;
  userId: string;
  lastReadAt: string;
};

export type NotificationType = "match" | "like" | "message";

export type AppNotification = {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  nextCursor: string | null;
  unreadCount: number;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";
