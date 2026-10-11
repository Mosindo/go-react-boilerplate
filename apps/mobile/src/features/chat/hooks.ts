import { useCallback, useMemo } from 'react';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { chatApi, queryKeys } from '../../api';
import type { Conversation, Message } from '../../api';
import { useErrorToast } from '../../lib/useErrorToast';
import { conversationsMarkedRead, totalUnread } from '../../realtime/cache';
import type { ConversationsData } from '../../realtime/cache';

export function useConversations() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 15_000,
  });
  const conversations = useMemo<Conversation[]>(() => {
    const seen = new Set<string>();
    const list: Conversation[] = [];
    for (const page of query.data?.pages ?? []) {
      for (const c of page.conversations) {
        if (!seen.has(c.id)) {
          seen.add(c.id);
          list.push(c);
        }
      }
    }
    return list;
  }, [query.data]);
  return { query, conversations, unreadTotal: totalUnread(query.data) };
}

export function useMessages(conversationId: string) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
  });
  const messages = useMemo<Message[]>(() => {
    const seen = new Set<string>();
    const list: Message[] = [];
    for (const page of query.data?.pages ?? []) {
      for (const m of page.messages) {
        if (!seen.has(m.id)) {
          seen.add(m.id);
          list.push(m);
        }
      }
    }
    return list;
  }, [query.data]);
  const otherLastReadAt = query.data?.pages[0]?.otherLastReadAt ?? null;
  return { query, messages, otherLastReadAt };
}

/** POST /conversations/:id/read and clear the local unread counter right away. */
export function useMarkRead(conversationId: string) {
  const qc = useQueryClient();
  return useCallback(() => {
    qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
      old ? conversationsMarkedRead(old, conversationId) : old,
    );
    chatApi.markRead(conversationId).catch(() => {
      // Best effort: the next focus / incoming message retries.
    });
  }, [qc, conversationId]);
}

export function useHideConversation() {
  const qc = useQueryClient();
  const onError = useErrorToast();
  return useMutation({
    mutationFn: (conversationId: string) => chatApi.hide(conversationId),
    onSuccess: (_data, conversationId) => {
      qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
        old
          ? {
              ...old,
              pages: old.pages.map((p) => ({
                ...p,
                conversations: p.conversations.filter((c) => c.id !== conversationId),
              })),
            }
          : old,
      );
    },
    onError,
  });
}
