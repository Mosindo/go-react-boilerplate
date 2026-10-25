import type { InfiniteData } from "@tanstack/react-query";
import type {
  AppNotification,
  ConversationItem,
  MatchItem,
  Message,
  NotificationsPage,
  Page,
  PublicProfile
} from "./api";

export function message(overrides: Partial<Message> = {}): Message {
  return {
    id: "m1",
    conversationId: "c1",
    senderId: "other",
    body: "Salut",
    createdAt: "2026-03-10T12:00:00.000Z",
    readAt: null,
    ...overrides
  };
}

export function conversation(overrides: Partial<ConversationItem> = {}): ConversationItem {
  return {
    id: "c1",
    matchId: "match1",
    otherUser: { userId: "other", firstName: "Camille", thumbUrl: null },
    lastMessage: message(),
    unreadCount: 0,
    updatedAt: "2026-03-10T12:00:00.000Z",
    ...overrides
  };
}

export function publicProfile(overrides: Partial<PublicProfile> = {}): PublicProfile {
  return {
    userId: "other",
    firstName: "Camille",
    age: 29,
    bio: "",
    city: "Lyon",
    distanceKm: 5,
    interests: [],
    photos: [
      { id: "p1", position: 0, isPrimary: true, url: "/photos/p1/image", thumbUrl: "/photos/p1/image?size=thumb" }
    ],
    sharedInterests: [],
    ...overrides
  };
}

export function matchItem(overrides: Partial<MatchItem> = {}): MatchItem {
  return {
    matchId: "match1",
    conversationId: "c1",
    createdAt: "2026-03-10T12:00:00.000Z",
    profile: publicProfile(),
    ...overrides
  };
}

export function notification(overrides: Partial<AppNotification> = {}): AppNotification {
  return {
    id: "n1",
    type: "message",
    actor: { userId: "other", firstName: "Camille", thumbUrl: null },
    conversationId: "c1",
    readAt: null,
    createdAt: "2026-03-10T12:00:00.000Z",
    ...overrides
  };
}

export function pages<T>(...items: T[][]): InfiniteData<Page<T>, string | undefined> {
  return {
    pages: items.map((pageItems, index) => ({
      items: pageItems,
      nextCursor: index < items.length - 1 ? `cursor-${index}` : null
    })),
    pageParams: items.map((_page, index) => (index === 0 ? undefined : `cursor-${index - 1}`))
  };
}

export function notificationPages(
  items: AppNotification[],
  unreadCount: number
): InfiniteData<NotificationsPage, string | undefined> {
  return { pages: [{ items, nextCursor: null, unreadCount }], pageParams: [undefined] };
}
