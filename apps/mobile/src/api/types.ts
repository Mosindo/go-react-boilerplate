export type Gender = "woman" | "man" | "nonbinary";
export type RelationshipGoal = "long_term" | "short_term" | "friendship" | "unsure";

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
  slug: string;
  label: string;
  category?: string;
};

export type PhotoView = {
  id: string;
  url: string;
  position: number;
  width: number;
  height: number;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type MissingField = "firstName" | "birthdate" | "gender" | "interestedIn" | "photos" | "location";

export type OwnProfile = {
  userId: string;
  firstName: string;
  birthdate: string | null;
  age: number | null;
  gender: Gender | null;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  city: string;
  hasLocation: boolean;
  isPaused: boolean;
  showDistance: boolean;
  isComplete: boolean;
  missing: MissingField[];
  photos: PhotoView[];
  interests: Interest[];
  preferences: Preferences;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal | null;
  city: string;
  distanceKm: number | null;
  photos: PhotoView[];
  interests: Interest[];
  sharedInterests: string[];
  isDemo: boolean;
};

export type UpdateProfileInput = Partial<{
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number;
  relationshipGoal: RelationshipGoal | "";
  interests: string[];
  isPaused: boolean;
  showDistance: boolean;
}>;

export type MatchInfo = {
  id: string;
  createdAt: string;
  user: PublicProfile;
};

export type SwipeAction = "like" | "pass";

export type SwipeResult = {
  action: SwipeAction;
  matched: boolean;
  match?: MatchInfo;
};

export type Message = {
  id: string;
  senderUserId: string;
  recipientUserId: string;
  content: string;
  createdAt: string;
};

export type Peer = {
  userId: string;
  firstName: string;
  age: number;
  photoUrl: string;
  isDemo: boolean;
};

export type Conversation = {
  matchId: string;
  matchedAt: string;
  user: Peer;
  lastMessage: Message | null;
  unreadCount: number;
};

export type MessagesPage = {
  messages: Message[];
  hasMore: boolean;
  peerLastReadAt: string | null;
};

export type AppNotification = {
  id: string;
  userId: string;
  type: "match" | "message" | string;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
  readAt?: string;
};

export type NotificationsPage = {
  notifications: AppNotification[];
  unreadCount: number;
};

export type ReportReason = "fake_profile" | "inappropriate_content" | "harassment" | "spam" | "underage" | "other";

export type BlockedUser = {
  userId: string;
  firstName: string;
  blockedAt: string;
};
