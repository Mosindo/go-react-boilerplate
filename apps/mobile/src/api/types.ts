export type Gender = "man" | "woman" | "nonbinary";

export type Account = {
  id: string;
  email: string;
  createdAt: string;
  profileComplete: boolean;
};

export type AuthResponse = {
  accessToken: string;
  refreshToken: string;
  user: Account;
};

export type Interest = { id: number; slug: string; label: string; category: string };

export type Photo = { id: string; position: number; width: number; height: number; url: string };

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city?: string;
  interestIds: number[];
  showDistance: boolean;
  discoverable: boolean;
};

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthDate: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  hasLocation: boolean;
  showDistance: boolean;
  discoverable: boolean;
  interests: Interest[];
  photos: Photo[];
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type ConversationUser = {
  userId: string;
  firstName: string;
  age: number;
  photoUrl: string | null;
};

export type Conversation = {
  id: string;
  matchedAt: string;
  user: ConversationUser;
  lastMessage: Message | null;
  unreadCount: number;
};

export type ConversationsPage = { conversations: Conversation[]; nextCursor: string | null };
export type MessagesPage = { messages: Message[]; nextCursor: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  conversationId: string;
  actor: { userId: string; firstName: string };
  readAt: string | null;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  nextCursor: string | null;
  unreadCount: number;
};

export type SwipeAction = "like" | "pass";
export type SwipeResult = { action: SwipeAction; matched: boolean; conversation?: Conversation };

export type BlockedUser = { userId: string; firstName: string; createdAt: string };

export type ReportReason =
  | "spam"
  | "fake_profile"
  | "harassment"
  | "inappropriate_content"
  | "underage"
  | "other";

export type ReportInput = {
  userId: string;
  reason: ReportReason;
  details?: string;
  conversationId?: string;
};
