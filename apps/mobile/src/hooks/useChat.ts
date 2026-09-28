import { useCallback, useRef } from "react";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../api/client";
import { getMessages, markConversationRead, sendMessage } from "../api/matches";
import { queryKeys } from "../api/queryKeys";
import type { ChatMessage } from "../api/types";
import {
  addMessage,
  applyMessageToMatches,
  clearUnread,
  replaceMessage,
  setMessageStatus,
  type MatchPages,
  type MessagePages
} from "../domain/realtimeCache";
import { showToast } from "../shared/feedback";

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => getMessages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor ?? undefined,
    staleTime: 0
  });
}

export function flattenMessages(data: MessagePages | undefined): ChatMessage[] {
  return data ? data.pages.flatMap((page) => page.items) : [];
}

/** Marks the thread read on the server and clears the local unread badge. */
export function useMarkRead(conversationId: string) {
  const queryClient = useQueryClient();
  return useCallback(async () => {
    queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => clearUnread(data, conversationId));
    try {
      await markConversationRead(conversationId);
    } catch {
      // Not critical: the next refetch restores the accurate unread count.
    }
  }, [conversationId, queryClient]);
}

/** Optimistic send. A failed message stays in the thread with a retry affordance. */
export function useSendMessage(conversationId: string, myUserId: string) {
  const queryClient = useQueryClient();
  const counter = useRef(0);
  const key = queryKeys.messages(conversationId);

  return useCallback(
    async (body: string, retryOf?: ChatMessage) => {
      const tempId = retryOf?.id ?? `local-${Date.now()}-${counter.current++}`;
      const optimistic: ChatMessage = retryOf
        ? { ...retryOf, status: "sending" }
        : {
            id: tempId,
            conversationId,
            senderId: myUserId,
            body,
            createdAt: new Date().toISOString(),
            readAt: null,
            status: "sending"
          };
      queryClient.setQueryData<MessagePages>(key, (data) => addMessage(data, optimistic));
      try {
        const saved = await sendMessage(conversationId, body);
        queryClient.setQueryData<MessagePages>(key, (data) => replaceMessage(data, tempId, saved));
        queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) =>
          applyMessageToMatches(data, saved, myUserId, conversationId)
        );
      } catch (error) {
        queryClient.setQueryData<MessagePages>(key, (data) => setMessageStatus(data, tempId, "failed"));
        showToast(
          error instanceof ApiError && error.code === "blocked"
            ? "You can't message this person."
            : errorMessage(error, "Message not sent."),
          "error"
        );
      }
    },
    [conversationId, key, myUserId, queryClient]
  );
}
