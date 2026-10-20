import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../api/keys";
import type { ConversationsPage, Message, MessagesPage, RealtimeEvent } from "../api/types";

export type RealtimeContext = { userId: string; activeConversationId: string | null };

type MessagesData = InfiniteData<MessagesPage, string | null>;
type ConversationsData = InfiniteData<ConversationsPage, string | null>;

export function upsertMessage(data: MessagesData | undefined, message: Message): MessagesData | undefined {
  if (!data || data.pages.length === 0) return data;
  const exists = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (exists) return data;
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

export function replaceMessage(
  data: MessagesData | undefined,
  tempId: string,
  message: Message
): MessagesData | undefined {
  if (!data) return data;
  const alreadyThere = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      messages: alreadyThere
        ? page.messages.filter((m) => m.id !== tempId)
        : page.messages.map((m) => (m.id === tempId ? message : m))
    }))
  };
}

export function removeMessage(data: MessagesData | undefined, id: string): MessagesData | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({ ...page, messages: page.messages.filter((m) => m.id !== id) }))
  };
}

export function setOtherLastRead(data: MessagesData | undefined, readAt: string): MessagesData | undefined {
  if (!data || data.pages.length === 0) return data;
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, otherLastReadAt: readAt }, ...rest] };
}

export function applyIncomingMessageToConversations(
  data: ConversationsData | undefined,
  message: Message,
  ctx: RealtimeContext
): { data: ConversationsData | undefined; found: boolean } {
  if (!data) return { data, found: false };
  let found = false;
  const incoming = message.senderId !== ctx.userId && ctx.activeConversationId !== message.conversationId;
  const pages = data.pages.map((page) => ({
    ...page,
    conversations: page.conversations.map((c) => {
      if (c.id !== message.conversationId) return c;
      found = true;
      return {
        ...c,
        lastMessage: message,
        updatedAt: message.createdAt,
        unreadCount: incoming ? c.unreadCount + 1 : c.unreadCount
      };
    })
  }));
  if (!found) return { data, found };
  // Most recently updated conversation first.
  const [first, ...rest] = pages;
  const all = pages.flatMap((p) => p.conversations);
  const target = all.find((c) => c.id === message.conversationId);
  if (!target) return { data, found };
  const reordered = [target, ...first.conversations.filter((c) => c.id !== target.id)];
  const cleanedRest = rest.map((p) => ({
    ...p,
    conversations: p.conversations.filter((c) => c.id !== target.id)
  }));
  return { data: { ...data, pages: [{ ...first, conversations: reordered }, ...cleanedRest] }, found: true };
}

export function clearConversationUnread(
  data: ConversationsData | undefined,
  id: string
): ConversationsData | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      conversations: page.conversations.map((c) => (c.id === id ? { ...c, unreadCount: 0 } : c))
    }))
  };
}

export function applyRealtimeEvent(qc: QueryClient, event: RealtimeEvent, ctx: RealtimeContext): void {
  switch (event.type) {
    case "message.new": {
      const { message } = event;
      qc.setQueryData<MessagesData>(queryKeys.messages(message.conversationId), (d) =>
        upsertMessage(d, message)
      );
      let found = false;
      qc.setQueryData<ConversationsData>(queryKeys.conversations, (d) => {
        const result = applyIncomingMessageToConversations(d, message, ctx);
        found = result.found;
        return result.data;
      });
      if (!found) void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      return;
    }
    case "message.read": {
      if (event.userId === ctx.userId) return;
      qc.setQueryData<MessagesData>(queryKeys.messages(event.conversationId), (d) =>
        setOtherLastRead(d, event.readAt)
      );
      return;
    }
    case "match.new": {
      void qc.invalidateQueries({ queryKey: queryKeys.matches });
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      void qc.invalidateQueries({ queryKey: queryKeys.notifications });
      return;
    }
    case "match.removed": {
      // Unmatch or block: drop the thread from caches and refetch the lists.
      qc.removeQueries({ queryKey: queryKeys.messages(event.conversationId) });
      void qc.invalidateQueries({ queryKey: queryKeys.matches });
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      return;
    }
    case "notification.new": {
      void qc.invalidateQueries({ queryKey: queryKeys.notifications });
      return;
    }
    default:
      return;
  }
}
