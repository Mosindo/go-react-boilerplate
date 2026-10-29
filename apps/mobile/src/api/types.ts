export type Gender = "man" | "woman" | "non_binary";

export type Me = {
  id: string;
  email: string;
  createdAt: string;
  hasProfile: boolean;
  hasPreferences: boolean;
  hasPhotos: boolean;
  onboardingComplete: boolean;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: Me;
};

export type Interest = { id: number; slug: string; label: string };

export type PhotoRef = { id: string; position: number; url: string };

export type Photo = PhotoRef & { width: number; height: number };

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  isVisible: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  latitude?: number;
  longitude?: number;
  interestIds: number[];
  isVisible?: boolean;
  showDistance?: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
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
  photos: PhotoRef[];
};

export type SwipeAction = "like" | "pass";

export type MatchUser = { id: string; firstName: string; photoUrl: string | null };

export type Match = { id: string; conversationId: string; user: MatchUser };

export type SwipeResult = {
  action: SwipeAction;
  alreadyProcessed: boolean;
  matched: boolean;
  match: Match | null;
};

export type Message = {
  id: number;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type Conversation = {
  id: string;
  matchId: string;
  matchedAt: string;
  user: MatchUser;
  lastMessage: Message | null;
  unreadCount: number;
  sortAt: string;
};

export type ConversationsPage = { conversations: Conversation[]; nextBefore: string | null };

export type MessagesPage = { messages: Message[]; nextBefore: number | null };

export type NotificationItem = {
  id: string;
  type: "match" | "message";
  data: Record<string, string>;
  readAt: string | null;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: NotificationItem[];
  unreadCount: number;
  nextBefore: string | null;
};

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "scam" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type RealtimeEvent =
  | { type: "ready" }
  | { type: "message"; data: Message }
  | { type: "match"; data: Match }
  | { type: "notification"; data: NotificationItem }
  | { type: "read"; data: { conversationId: string; readerId: string; readAt: string } }
  | { type: "unmatch"; data: { matchId: string; conversationId: string } };
