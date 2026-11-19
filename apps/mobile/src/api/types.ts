export type Gender = "woman" | "man" | "nonbinary";
export type LookingFor = "relationship" | "casual" | "friendship" | "unsure";

export type AuthUser = { id: string; email: string; createdAt: string };

export type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

export type Interest = { id: number; slug: string; label: string };

export type Photo = { id: string; url: string; position: number; width: number; height: number };

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
  isDiscoverable: boolean;
  showDistance: boolean;
};

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  lookingFor: LookingFor;
  city: string;
  hasLocation: boolean;
  interests: Interest[];
  photos: Photo[];
  isComplete: boolean;
  preferences: Preferences;
};

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  lookingFor: LookingFor;
  city: string;
  interestIds: number[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  lookingFor: LookingFor;
  city: string;
  distanceKm: number | null;
  interests: Interest[];
  photos: Photo[];
};

export type Match = { id: string; conversationId: string; user: ProfileCard };

export type SwipeResponse = { matched: boolean; match?: Match };

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  isMine: boolean;
  isRead: boolean;
};

export type Partner = { userId: string; firstName: string; age: number; photoUrl: string | null };

export type Conversation = {
  id: string;
  matchId: string;
  matchedAt: string;
  partner: Partner;
  lastMessage: Message | null;
  unreadCount: number;
};

export type ConversationsPage = { conversations: Conversation[]; nextCursor: string | null };

export type MessagesPage = { messages: Message[]; nextCursor: string | null; partnerLastReadAt: string | null };

export type AppNotification = {
  id: string;
  type: "match" | "message";
  actor: { userId: string; firstName: string; photoUrl: string | null } | null;
  conversationId: string | null;
  readAt: string | null;
  createdAt: string;
};

export type NotificationsPage = { notifications: AppNotification[]; nextCursor: string | null; unreadCount: number };

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };
