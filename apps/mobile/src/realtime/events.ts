import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { keys } from "../api/keys";
import type { Message, MessagesPage, RealtimeEvent } from "../api/types";

/** Parses a WebSocket frame; unknown or malformed frames are ignored. */
export function parseEvent(raw: string): RealtimeEvent | null {
  try {
    const value = JSON.parse(raw) as { type?: string; data?: unknown };
    switch (value.type) {
      case "message.new":
      case "conversation.read":
      case "match.new":
      case "match.removed":
        return value as RealtimeEvent;
      default:
        return null;
    }
  } catch {
    return null;
  }
}

function withMessage(data: InfiniteData<MessagesPage> | undefined, message: Message): InfiniteData<MessagesPage> | undefined {
  if (!data || data.pages.length === 0) return data;
  if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) return data;
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

/** Keeps the React Query cache in sync with a pushed event. */
export function applyRealtimeEvent(client: QueryClient, event: RealtimeEvent): void {
  switch (event.type) {
    case "message.new": {
      client.setQueryData<InfiniteData<MessagesPage>>(keys.messages(event.data.conversationId), (data) => withMessage(data, event.data));
      void client.invalidateQueries({ queryKey: keys.conversations });
      void client.invalidateQueries({ queryKey: keys.unread });
      void client.invalidateQueries({ queryKey: keys.matches });
      break;
    }
    case "conversation.read": {
      client.setQueryData<InfiniteData<MessagesPage>>(keys.messages(event.data.conversationId), (data) => {
        if (!data || data.pages.length === 0) return data;
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, otherLastReadAt: event.data.readAt }, ...rest] };
      });
      break;
    }
    case "match.new":
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.notifications });
      break;
    case "match.removed":
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.conversations });
      void client.invalidateQueries({ queryKey: keys.unread });
      void client.invalidateQueries({ queryKey: keys.notifications });
      break;
  }
}

/** Exponential backoff with a ceiling, used between reconnect attempts. */
export function reconnectDelay(attempt: number): number {
  return Math.min(30_000, 1_000 * 2 ** Math.min(attempt, 5));
}
