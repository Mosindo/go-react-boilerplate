export type Gender = "woman" | "man" | "nonbinary";

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
};

export type Photo = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type ProfileInfo = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  profile: ProfileInfo | null;
  preferences: Preferences | null;
  interests: Interest[];
  photos: Photo[];
  hasLocation: boolean;
  locationUpdatedAt: string | null;
  discoverable: boolean;
  showDistance: boolean;
  isComplete: boolean;
};

export type ProfileInput = Omit<ProfileInfo, "age">;

export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  distanceKm: number | null;
  interests: (Interest & { shared: boolean })[];
  photos: Photo[];
};

export type SwipeDirection = "like" | "pass";

export type MatchPayload = {
  matchId: string;
  createdAt: string;
  user: Card;
};

export type SwipeResult = {
  matched: boolean;
  match?: MatchPayload;
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
  createdAt: string;
  user: Card;
  lastMessage: Message | null;
  unreadCount: number;
  otherLastReadAt: string | null;
};

export type ConversationsPage = {
  conversations: Conversation[];
  nextCursor: string;
};

export type MessagesPage = {
  messages: Message[];
  nextCursor: string;
  otherLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { matchId?: string };
  isRead: boolean;
  createdAt: string;
  readAt: string | null;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
  hasMore: boolean;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};

export type RealtimeEvent =
  | { type: "message.created"; data: { message: Message } }
  | { type: "conversation.read"; data: { matchId: string; userId: string; readAt: string } }
  | { type: "match.created"; data: MatchPayload }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.created"; data: AppNotification };
