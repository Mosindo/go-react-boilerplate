/** Types mirroring docs/API.md exactly. */

export type Gender = 'man' | 'woman' | 'non_binary';
export const GENDERS: readonly Gender[] = ['man', 'woman', 'non_binary'];

export interface Interest {
  id: number;
  slug: string;
  label: string;
}

export interface Photo {
  id: string;
  position: number;
  isPrimary: boolean;
  /** Relative to the API base URL, signed + expiring, no auth header needed. */
  url: string;
}

export interface PublicProfile {
  userId: string;
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  interests: Interest[];
  photos: Photo[];
  /** Approximate; null when unknown or hidden by the other user. */
  distanceKm: number | null;
  likedYou?: boolean;
}

export interface Preferences {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  /** 1-500, or null for "anywhere". */
  maxDistanceKm: number | null;
}

export interface OwnProfile extends Omit<PublicProfile, 'distanceKm'> {
  /** YYYY-MM-DD */
  birthDate: string;
  discoverable: boolean;
  showDistance: boolean;
  hasLocation: boolean;
  preferences: Preferences;
}

export interface Privacy {
  discoverable: boolean;
  showDistance: boolean;
}

export interface Message {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
}

export interface ConversationUser {
  userId: string;
  firstName: string;
  photoUrl: string | null;
}

export interface LastMessage {
  body: string;
  senderId: string;
  createdAt: string;
}

export interface Conversation {
  id: string;
  matchId: string;
  user: ConversationUser;
  lastMessage: LastMessage | null;
  unreadCount: number;
  createdAt: string;
  lastActivityAt: string;
}

export type NotificationType = 'new_like' | 'new_match';

export interface AppNotification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  data: Record<string, string>;
  isRead: boolean;
  createdAt: string;
}

/* ---------------------------------- Auth ---------------------------------- */

export interface User {
  id: string;
  email: string;
  createdAt: string;
  profileComplete: boolean;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  user: User;
}

export interface Credentials {
  email: string;
  password: string;
}

export interface ChangePasswordInput {
  currentPassword: string;
  newPassword: string;
}

export interface ResetPasswordInput {
  token: string;
  password: string;
}

/* --------------------------------- Profile -------------------------------- */

export interface ProfileInput {
  firstName: string;
  /** YYYY-MM-DD; can't change once set. */
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interestIds: number[];
}

export interface LocationInput {
  latitude: number;
  longitude: number;
}

/* ------------------------------ Swipes / pages ----------------------------- */

export type SwipeAction = 'like' | 'pass';

export interface SwipeResult {
  action: SwipeAction;
  matched: boolean;
  conversation: Conversation | null;
}

export interface ConversationsPage {
  conversations: Conversation[];
  nextCursor: string | null;
}

export interface MessagesPage {
  /** Newest first. */
  messages: Message[];
  nextCursor: string | null;
  otherLastReadAt: string | null;
}

export interface NotificationsPage {
  notifications: AppNotification[];
  unreadCount: number;
  nextCursor: string | null;
}

/* --------------------------------- Safety --------------------------------- */

export interface BlockedUser {
  userId: string;
  firstName: string;
  createdAt: string;
}

export const REPORT_REASONS = [
  'spam',
  'fake_profile',
  'inappropriate_content',
  'harassment',
  'underage',
  'scam',
  'other',
] as const;
export type ReportReason = (typeof REPORT_REASONS)[number];

export interface ReportInput {
  userId: string;
  reason: ReportReason;
  details?: string;
}

/* --------------------------------- Realtime -------------------------------- */

export interface WsTicket {
  ticket: string;
  expiresInSeconds: number;
}

export interface MessageReadEvent {
  conversationId: string;
  readerId: string;
  readAt: string;
}

export type RealtimeEvent =
  | { type: 'message.new'; data: Message }
  | { type: 'message.read'; data: MessageReadEvent }
  | { type: 'match.new'; data: Conversation }
  | { type: 'notification.new'; data: AppNotification }
  | { type: 'conversation.removed'; data: { conversationId: string } };

/* ---------------------------------- Errors --------------------------------- */

export type ApiErrorCode =
  | 'invalid_request'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'conflict'
  | 'rate_limited'
  | 'profile_required'
  | 'profile_not_found'
  | 'invalid_credentials'
  | 'email_taken'
  | 'underage'
  | 'photo_limit'
  | 'invalid_image'
  | 'blocked';

export interface ApiErrorBody {
  error: string;
  code: string;
}

/** Local-only codes produced by the client itself. */
export type ClientErrorCode = 'network' | 'timeout' | 'unknown';
