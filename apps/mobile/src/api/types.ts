// Shapes returned by the API. Keep in sync with services/api (see README "API overview").

export type Gender = "man" | "woman" | "nonbinary";

export type Interest = {
  slug: string;
  label: string;
};

export type PhotoView = {
  id: string;
  /** Relative signed path; resolve it with mediaUrl(). */
  url: string;
  thumbUrl: string;
  position: number;
  width: number;
  height: number;
};

/** Public view of another person. It never contains exact coordinates. */
export type Card = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city?: string;
  /** Approximate distance, rounded up to a multiple of 5 km. */
  distanceKm?: number;
  interests: Interest[];
  commonInterests: number;
  photos: PhotoView[];
  likedMe: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city?: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoView[];
  preferences?: Preferences;
};

export type Onboarding = {
  profile: boolean;
  preferences: boolean;
  photos: boolean;
  complete: boolean;
};

export type MyProfileResponse = {
  profile: MyProfile | null;
  onboarding: Onboarding;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  interests: string[];
};

export type Peer = {
  userId: string;
  firstName: string;
  thumbUrl?: string;
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
  peer: Peer;
  lastMessage?: Message;
  unreadCount: number;
  matchedAt: string;
  updatedAt: string;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  peerLastReadAt?: string;
};

export type NotificationType = "match" | "message";

export type AppNotification = {
  id: string;
  type: NotificationType | string;
  title: string;
  body: string;
  data: {
    conversationId?: string;
    userId?: string;
    firstName?: string;
    matchId?: string;
  };
  isRead: boolean;
  createdAt: string;
};

export type NotificationsResponse = {
  notifications: AppNotification[];
  unreadCount: number;
};

export type MatchInfo = {
  id: string;
  conversationId: string;
  userId: string;
  firstName: string;
};

export type SwipeAction = "like" | "pass";

export type SwipeResponse = {
  matched: boolean;
  match?: MatchInfo;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  thumbUrl?: string;
  blockedAt: string;
};

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readAt: string } }
  | { type: "match.new"; data: MatchInfo }
  | { type: "match.removed"; data: { userId: string; conversationId: string } }
  | { type: "notification.new"; data: AppNotification };
