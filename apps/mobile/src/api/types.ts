export type Gender = "woman" | "man" | "nonbinary";

export type Interest = { slug: string; label: string };
export type PhotoRef = { id: string; url: string };
export type Location = { latitude: number; longitude: number };

/** Public profile shown to other users: never contains exact location data. */
export type Card = {
  id: string;
  firstName: string;
  age: number;
  gender: Gender | "";
  bio: string;
  city: string;
  distanceKm?: number;
  interests: Interest[];
  photos: PhotoRef[];
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type OwnProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender | "";
  bio: string;
  city: string;
  location: Location | null;
  isDiscoverable: boolean;
  showDistance: boolean;
  interests: Interest[];
  photos: PhotoRef[];
  preferences: Preferences;
  complete: boolean;
  exists: boolean;
};

export type ProfileInput = {
  firstName: string;
  gender: Gender | "";
  bio: string;
  city: string;
  location: Location | null;
  interests: string[];
  isDiscoverable: boolean;
  showDistance: boolean;
};

export type MatchItem = {
  id: string;
  conversationId: string;
  createdAt: string;
  user: Card;
};

export type SwipeResult = {
  matched: boolean;
  alreadyHandled: boolean;
  match?: MatchItem;
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  /** Client-only: set while a message is being sent. */
  pending?: boolean;
};

export type MessagesPage = {
  messages: Message[];
  nextBefore?: string;
  otherLastReadAt?: string;
};

export type Conversation = {
  id: string;
  matchId: string;
  user: Card;
  lastMessage?: { body: string; senderId: string; createdAt: string };
  unreadCount: number;
  createdAt: string;
  sortAt: string;
};

export type NotificationItem = {
  id: string;
  type: "match" | "message" | "like";
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
};

export type NotificationsPage = {
  notifications: NotificationItem[];
  unreadCount: number;
  nextBefore?: string;
};

export type BlockedUser = { id: string; firstName: string; blockedAt: string };

export type ReportReason = "spam" | "fake" | "harassment" | "inappropriate" | "underage" | "other";

export type AuthUser = { id: string; email: string; birthDate: string; createdAt: string };

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};
