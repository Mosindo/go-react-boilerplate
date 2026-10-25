import { useRef } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, queryKeys, type ConversationItem, type Message } from "../../api";
import {
  appendMessage,
  applyMessageToConversations,
  clearConversationUnread,
  createLocalId,
  markIncomingRead,
  removeMessage,
  replaceMessage,
  type Paged
} from "../../realtime/cache";
import { cursorQuery } from "../../shared/pagination";

export function useThread(conversationId: string) {
  return useInfiniteQuery({
    ...cursorQuery(queryKeys.messages(conversationId), (cursor) => chatApi.messages(conversationId, cursor)),
    staleTime: 15_000
  });
}

export function useUnreadMessageCount() {
  return useQuery({ queryKey: queryKeys.unreadCount, queryFn: chatApi.unreadCount, staleTime: 30_000 });
}

/** Sends a message with an optimistic bubble that is swapped for the server copy (or dropped on failure). */
export function useSendMessage(conversationId: string, currentUserId: string, onFailure: (body: string) => void) {
  const queryClient = useQueryClient();
  const sequence = useRef(0);
  const threadKey = queryKeys.messages(conversationId);

  return useMutation({
    mutationFn: (body: string) => chatApi.send(conversationId, body),
    onMutate: async (body) => {
      await queryClient.cancelQueries({ queryKey: threadKey });
      sequence.current += 1;
      const optimistic: Message = {
        id: createLocalId(sequence.current),
        conversationId,
        senderId: currentUserId,
        body,
        createdAt: new Date().toISOString(),
        readAt: null
      };
      queryClient.setQueryData<Paged<Message>>(threadKey, (data) => appendMessage(data, optimistic));
      return { localId: optimistic.id };
    },
    onSuccess: (message, _body, context) => {
      queryClient.setQueryData<Paged<Message>>(threadKey, (data) => replaceMessage(data, context.localId, message));
      let found = true;
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) => {
        const result = applyMessageToConversations(data, message, { currentUserId, isActive: true });
        found = result.found;
        return result.data;
      });
      if (!found) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (_error, body, context) => {
      if (context) {
        queryClient.setQueryData<Paged<Message>>(threadKey, (data) => removeMessage(data, context.localId));
      }
      onFailure(body);
    }
  });
}

/** Marks the other person's messages as read, then reflects it locally. */
export function useMarkRead(conversationId: string, currentUserId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => chatApi.markRead(conversationId),
    meta: { silent: true },
    onSuccess: () => {
      const readAt = new Date().toISOString();
      queryClient.setQueryData<Paged<Message>>(queryKeys.messages(conversationId), (data) =>
        markIncomingRead(data, currentUserId, readAt)
      );
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
        clearConversationUnread(data, conversationId)
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    }
  });
}
