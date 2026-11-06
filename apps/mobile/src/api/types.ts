export type Gender = "woman" | "man" | "nonbinary";

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type Interest = { slug: string; label: string };
export type Photo = { id: string; position: number; url: string };

export type OwnProfile = {
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  discoverable: boolean;
  showDistance: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MyProfile = {
  profile: OwnProfile | null;
  preferences: Preferences;
  interests: Interest[];
  photos: Photo[];
  onboardingComplete: boolean;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude: number;
  longitude: number;
  interests: string[];
};

export type PublicProfile = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = {
  matched: boolean;
  duplicate: boolean;
  matchId?: string;
  conversationId?: string;
};

export type Peer = { id: string; firstName: string; mainPhotoId: string | null };
export type LastMessage = { body: string; senderId: string; createdAt: string };
export type Conversation = {
  id: string;
  matchId: string;
  peer: Peer;
  lastMessage: LastMessage | null;
  unreadCount: number;
  activityAt: string;
};
export type ConversationsPage = {
  conversations: Conversation[];
  unreadTotal: number;
  nextBefore?: string;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type MessagesPage = {
  messages: Message[]; // newest first
  peerLastReadAt: string;
  peer: Peer;
  nextBefore?: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};
export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
  nextBefore?: string;
};

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };
export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; readAt: string } }
  | {
      type: "match.new";
      data: { matchId: string; conversationId: string; userId: string; firstName: string };
    }
  | { type: "match.removed"; data: Record<string, string> }
  | { type: "notification.new"; data: AppNotification };
