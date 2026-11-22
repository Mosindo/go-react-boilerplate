export type Gender = "woman" | "man" | "nonbinary";

export type AuthUser = {
  id: string;
  email: string;
  organizationId: string;
  createdAt: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type Interest = {
  id: number;
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

export type Preferences = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

export type MissingField = "firstName" | "birthdate" | "gender" | "interestedIn" | "location" | "photos";

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate?: string;
  age?: number;
  gender?: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  hasLocation: boolean;
  discoverable: boolean;
  showDistance: boolean;
  moderationStatus: "active" | "under_review" | "suspended";
  interests: Interest[];
  photos: Photo[];
  preferences: Preferences;
  isComplete: boolean;
  missingFields: MissingField[];
};

export type ProfileCard = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: Photo[];
  isDemo: boolean;
};

export type ProfilePatch = Partial<{
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  interests: string[];
  discoverable: boolean;
  showDistance: boolean;
}>;

export type DiscoveryStatus = "ok" | "incomplete_profile" | "paused" | "under_review";

export type DiscoveryResponse = {
  profiles: ProfileCard[];
  status: DiscoveryStatus;
  missingFields?: MissingField[];
};

export type Match = {
  id: string;
  matchedAt: string;
  conversationId?: string;
  user: ProfileCard;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  match?: Match;
};

export type ChatSummary = {
  matchId: string;
  conversationId: string;
  matchedAt: string;
  user: {
    id: string;
    firstName: string;
    photo?: Photo;
    isDemo: boolean;
  };
  lastMessage?: {
    id: string;
    senderUserId: string;
    content: string;
    createdAt: string;
  };
  unreadCount: number;
};

export type ChatsResponse = {
  chats: ChatSummary[];
  totalUnread: number;
};

export type Message = {
  id: string;
  conversationId: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
  /** Client-only: optimistic message waiting for the server. */
  pending?: boolean;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  otherLastReadAt?: string;
};

export type Notification = {
  id: string;
  type: string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsResponse = {
  notifications: Notification[];
  unreadCount: number;
};

export type BlockedUser = {
  userId: string;
  firstName: string;
  photo?: Photo;
  blockedAt: string;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";
