import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import * as chatApi from "../api/chat";
import * as discoverApi from "../api/discover";
import { queryKeys } from "../api/queryKeys";
import { applyRealtimeEvent } from "../api/realtimeCache";
import type { ChatMessage, MatchSummary } from "../api/types";
import { addLocalMessage, removeMessage, resolveLocalMessage, setLocalStatus } from "../lib/messageCache";
import { flattenPages, mapPages, nextCursorOf, removeItems, type CursorData } from "../lib/pagination";
import { getActiveConversation } from "../store/activeConversation";
import { useCurrentUserId } from "./useAuth";

export function useConversations() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.fetchConversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: nextCursorOf
  });
  return { ...query, items: flattenPages(query.data, "conversations") };
}

export function useMatches() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.matches,
    queryFn: ({ pageParam }) => discoverApi.fetchMatches(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: nextCursorOf
  });
  return { ...query, items: flattenPages(query.data, "matches") };
}

export function useConversationUnreadCount() {
  return (
    useQuery({
      queryKey: queryKeys.conversationsUnread,
      queryFn: chatApi.fetchConversationUnreadCount,
      staleTime: 15_000
    }).data ?? 0
  );
}

export function useMessages(conversationId: string) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.fetchMessages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: nextCursorOf
  });
  // API returns newest first; an inverted list wants exactly that order.
  return { ...query, items: flattenPages<"messages", ChatMessage>(query.data, "messages") };
}

type ConversationsData = CursorData<"conversations", MatchSummary>;
type MatchesData = CursorData<"matches", MatchSummary>;

function clearUnreadLocally(client: QueryClient, conversationId: string) {
  const before = client.getQueryData<ConversationsData>(queryKeys.conversations);
  const row = before?.pages.flatMap((p) => p.conversations).find((c) => c.conversationId === conversationId);
  client.setQueryData<ConversationsData>(queryKeys.conversations, (data) =>
    data
      ? mapPages(data, "conversations", (items) =>
          items.map((c) => (c.conversationId === conversationId ? { ...c, unreadCount: 0 } : c))
        )
      : data
  );
  if (row && row.unreadCount > 0) {
    client.setQueryData<number>(queryKeys.conversationsUnread, (count) => Math.max(0, (count ?? 0) - row.unreadCount));
  }
}

export function useMarkConversationRead(conversationId: string) {
  const client = useQueryClient();
  return useCallback(async () => {
    clearUnreadLocally(client, conversationId);
    try {
      await chatApi.markConversationRead(conversationId);
    } finally {
      void client.invalidateQueries({ queryKey: queryKeys.conversationsUnread });
    }
  }, [client, conversationId]);
}

/** Optimistic send with failed/retry state. */
export function useSendMessage(conversationId: string) {
  const client = useQueryClient();
  const myId = useCurrentUserId();

  const deliver = useCallback(
    async (localId: string, body: string) => {
      setLocalStatus(client, conversationId, localId, "sending");
      try {
        const saved = await chatApi.sendMessage(conversationId, body);
        resolveLocalMessage(client, localId, saved);
        // Keeps the conversation list / previews in sync (same path as a realtime echo).
        applyRealtimeEvent(
          client,
          { type: "message.new", data: saved },
          {
            myUserId: myId,
            getActiveConversationId: getActiveConversation
          }
        );
      } catch {
        setLocalStatus(client, conversationId, localId, "failed");
      }
    },
    [client, conversationId, myId]
  );

  const send = useCallback(
    (body: string) => {
      const trimmed = body.trim();
      if (!trimmed) return;
      const local = addLocalMessage(client, conversationId, myId, trimmed);
      void deliver(local.id, trimmed);
    },
    [client, conversationId, deliver, myId]
  );

  const retry = useCallback(
    (message: ChatMessage) => {
      void deliver(message.id, message.body);
    },
    [deliver]
  );

  const discard = useCallback(
    (message: ChatMessage) => removeMessage(client, conversationId, message.id),
    [client, conversationId]
  );

  return { send, retry, discard };
}

function dropConversationFromCaches(client: QueryClient, conversationId: string, matchId?: string) {
  client.setQueryData<ConversationsData>(queryKeys.conversations, (data) =>
    removeItems(data, "conversations", (c) => c.conversationId === conversationId || (!!matchId && c.id === matchId))
  );
  client.removeQueries({ queryKey: queryKeys.messages(conversationId) });
  void client.invalidateQueries({ queryKey: queryKeys.conversationsUnread });
}

export function useHideConversation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (conversationId: string) => chatApi.hideConversation(conversationId),
    onSuccess: (_v, conversationId) => dropConversationFromCaches(client, conversationId)
  });
}

export function useUnmatch() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (match: Pick<MatchSummary, "id" | "conversationId">) => discoverApi.unmatch(match.id),
    onSuccess: (_v, match) => {
      dropConversationFromCaches(client, match.conversationId, match.id);
      client.setQueryData<MatchesData>(queryKeys.matches, (data) =>
        removeItems(data, "matches", (m) => m.id === match.id)
      );
    }
  });
}
