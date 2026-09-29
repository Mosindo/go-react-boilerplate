/** Wire types shared by every API module. Mirrors docs/API.md — keep the two in sync. */

export type Gender = "woman" | "man" | "non_binary";

export type Photo = {
  id: string;
  /** Relative signed URL. Always pass it through `resolvePhotoUrl` before rendering. */
  url: string;
  position: number;
};

export type PublicProfile = {
  userId: string;
  firstName: string;
  age: number | null;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;
  interests: string[];
  photos: Photo[];
};

export type Message = {
  id: string;
  conversationId: string;
  senderId: string;
  body: string;
  createdAt: string;
  readAt: string | null;
};

export type ConversationSummary = {
  id: string;
  matchId: string;
  matchedAt: string;
  user: { userId: string; firstName: string; age: number | null; photo: Photo | null };
  lastMessage: Message | null;
  unreadCount: number;
  updatedAt: string;
};

export type AppNotification = {
  id: string;
  type: "match" | "message";
  title: string;
  body: string;
  data: { conversationId?: string; userId?: string };
  isRead: boolean;
  createdAt: string;
};

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "scam" | "other";

/** Events pushed over the WebSocket (see docs/API.md, "Realtime"). */
export type RealtimeEvent =
  | { type: "message.new"; data: { message: Message } }
  | { type: "conversation.read"; data: { conversationId: string; readAt: string } }
  | { type: "match.new"; data: { conversation: ConversationSummary } }
  | { type: "match.removed"; data: { conversationId: string } }
  | { type: "notification.new"; data: { notification: AppNotification } };
