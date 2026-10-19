export type Gender = "man" | "woman" | "non_binary";

export type Interest = { id: number; slug: string; label: string };
export type PhotoRef = { id: string; position: number };

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  bio: string;
  city?: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number | null;
  showDistance: boolean;
};

export type OwnProfile = PublicProfile & {
  birthDate: string;
  gender: Gender;
  isVisible: boolean;
  hasLocation: boolean;
  preferences: Preferences | null;
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  location?: { lat: number; lng: number } | null;
  interests: number[];
  isVisible?: boolean;
};

export type AuthUser = { id: string; email: string; createdAt: string };
export type AuthSession = { accessToken: string; refreshToken: string; user: AuthUser };

export type Match = { id: string; conversationId: string; createdAt: string; profile: PublicProfile };
export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; duplicate: boolean; match?: Match };

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
  createdAt: string;
  updatedAt: string;
  profile: PublicProfile;
  lastMessage?: Message;
  unreadCount: number;
};

export type MessagesPage = { messages: Message[]; peerLastReadAt: string; nextCursor?: string };
export type ConversationsPage = { conversations: Conversation[]; nextCursor?: string };

export type NotificationItem = {
  id: string;
  type: "new_match" | "new_message";
  actor?: { userId: string; firstName: string; photoId?: string };
  conversationId?: string;
  readAt?: string;
  createdAt: string;
};
export type NotificationsPage = {
  notifications: NotificationItem[];
  unreadCount: number;
  nextCursor?: string;
};

export type BlockedUser = { userId: string; firstName: string; photoId?: string; blockedAt: string };

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type LikedBy = PublicProfile & { likedAt: string };
