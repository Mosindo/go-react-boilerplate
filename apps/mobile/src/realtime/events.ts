import type { QueryClient } from "@tanstack/react-query";
import type { MatchesPage, MessagesPage, NotificationsPage, ServerEvent } from "../api/types";
import { queryKeys } from "../session/queryKeys";
import { activeChat } from "./activeChat";
import {
  appendMessage,
  applyMatchNew,
  applyMatchRemoved,
  applyMessageNewToMatches,
  markMessagesRead,
  prependNotification,
  type Paged
} from "./cache";

const EVENT_TYPES = new Set([
  "message.new",
  "message.read",
  "match.new",
  "match.removed",
  "notification.new"
]);

/** Parses a raw frame; returns null for control frames (ready/pong) and malformed input. */
export function parseServerEvent(raw: unknown): ServerEvent | null {
  if (typeof raw !== "string") return null;
  try {
    const parsed = JSON.parse(raw) as { type?: unknown; data?: unknown };
    if (typeof parsed.type !== "string" || !EVENT_TYPES.has(parsed.type)) return null;
    if (!parsed.data || typeof parsed.data !== "object") return null;
    return parsed as ServerEvent;
  } catch {
    return null;
  }
}

export type EventContext = {
  myUserId: string | null;
  /** Conversation currently on screen (its unread counter must not grow). */
  activeMatchId: string | null;
  now?: () => string;
};

/** Applies a realtime event to the React Query cache. */
export function applyServerEvent(
  queryClient: QueryClient,
  event: ServerEvent,
  context: EventContext
): void {
  switch (event.type) {
    case "message.new": {
      const message = event.data;
      queryClient.setQueryData<Paged<MessagesPage>>(queryKeys.messages(message.matchId), (data) =>
        appendMessage(data, message)
      );
      const matches = queryClient.getQueryData<Paged<MatchesPage>>(queryKeys.matches);
      const next = applyMessageNewToMatches(
        matches,
        message,
        context.myUserId,
        context.activeMatchId
      );
      if (next === null) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      } else if (next) {
        queryClient.setQueryData(queryKeys.matches, next);
      }
      return;
    }
    case "message.read": {
      const { matchId, readerId, readAt } = event.data;
      queryClient.setQueryData<Paged<MessagesPage>>(queryKeys.messages(matchId), (data) =>
        markMessagesRead(data, readerId, readAt)
      );
      return;
    }
    case "match.new": {
      const match = event.data;
      queryClient.setQueryData<Paged<MatchesPage>>(queryKeys.matches, (data) =>
        applyMatchNew(data, match)
      );
      if (!queryClient.getQueryData(queryKeys.matches)) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      }
      return;
    }
    case "match.removed": {
      const { matchId } = event.data;
      queryClient.setQueryData<Paged<MatchesPage>>(queryKeys.matches, (data) =>
        applyMatchRemoved(data, matchId)
      );
      queryClient.removeQueries({ queryKey: queryKeys.messages(matchId) });
      activeChat.emitRemoved(matchId);
      return;
    }
    case "notification.new": {
      const notification = event.data;
      queryClient.setQueryData<Paged<NotificationsPage>>(queryKeys.notifications, (data) =>
        prependNotification(data, notification)
      );
      if (!queryClient.getQueryData(queryKeys.notifications)) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      }
      return;
    }
  }
}

/** After a (re)connect the cache may have missed events: refetch the lists. */
export function refetchAfterReconnect(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
  void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  void queryClient.invalidateQueries({ queryKey: ["messages"] });
}
