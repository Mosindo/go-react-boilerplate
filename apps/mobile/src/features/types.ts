// Contract types for discovery, chat, notifications and safety: re-exports src/api/types.ts and adds the request/event shapes.

import type {
  Conversation,
  Gender,
  Interest,
  Match,
  Message,
  Notification,
  NotificationType,
  Photo,
  PublicProfile,
  UserSummary
} from "../api/types";

export type {
  Conversation,
  Gender,
  Interest,
  Match,
  Message,
  NotificationType,
  Photo,
  PublicProfile,
  UserSummary
};
export type AppNotification = Notification;

export type SwipeAction = "like" | "pass";
export type SwipeResult = { matched: boolean; match: Match | null };

export type MessagesPage = { items: Message[]; nextCursor: string | null };
export type NotificationsPayload = { items: AppNotification[]; unreadCount: number };

export type ReportReason =
  "spam" | "fake_profile" | "harassment" | "inappropriate_content" | "underage" | "other";

export type ReadEvent = { conversationId: string; readerId: string; readAt: string };

export type RealtimeEvent =
  | { type: "message"; data: Message }
  | { type: "read"; data: ReadEvent }
  | { type: "match"; data: Match }
  | { type: "notification"; data: AppNotification };

export type RealtimeEventType = RealtimeEvent["type"];
export type RealtimeEventData<T extends RealtimeEventType> = Extract<RealtimeEvent, { type: T }>["data"];
