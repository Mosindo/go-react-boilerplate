export type Gender = "man" | "woman" | "non_binary";
export type LookingFor = "relationship" | "casual" | "friends" | "open";

export type Interest = { slug: string; label: string };
export type Photo = { id: string; position: number; url: string; thumbUrl: string };

export type AccountInfo = { id: string; email: string; createdAt: string };

export type OwnProfile = {
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  occupation: string;
  lookingFor: LookingFor;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
  isVisible: boolean;
  showDistance: boolean;
};

export type Onboarding = { hasProfile: boolean; hasPhoto: boolean; complete: boolean };

export type Me = {
  user: AccountInfo;
  profile: OwnProfile | null;
  preferences: Preferences;
  photos: Photo[];
  onboarding: Onboarding;
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  occupation: string;
  lookingFor: LookingFor;
  city: string;
  interests: string[];
};

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
  interests: Interest[];
  sharedInterests: number;
  photos: Photo[];
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  alreadyProcessed: boolean;
  matchId?: string;
  conversationId?: string;
};

export type MatchItem = {
  id: string;
  conversationId: string;
  createdAt: string;
  hasMessages: boolean;
  user: PublicProfile;
};
export type MatchesPage = { matches: MatchItem[]; nextCursor?: string };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
};
export type MessagesPage = { messages: Message[]; otherLastReadAt: string; nextCursor?: string };

export type ConversationItem = {
  id: string;
  matchId: string;
  user: PublicProfile;
  lastMessage: Message | null;
  lastMessageAt: string;
  unreadCount: number;
};
export type ConversationsPage = { conversations: ConversationItem[]; nextCursor?: string };
export type UnreadSummary = { unreadMessages: number; unreadConversations: number };

export type AppNotification = {
  id: string;
  type: "match" | "system";
  title: string;
  body: string;
  data: { matchId?: string; conversationId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};
export type NotificationsPage = { notifications: AppNotification[]; nextCursor?: string };

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportReason = "spam" | "harassment" | "fake_profile" | "inappropriate_content" | "underage" | "other";

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type RealtimeEvent =
  | { type: "message.new"; data: Message }
  | { type: "conversation.read"; data: { conversationId: string; readAt: string } }
  | { type: "match.new"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "match.removed"; data: { matchId: string } };
