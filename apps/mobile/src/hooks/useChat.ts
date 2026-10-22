import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { fetchMessages, markConversationRead, sendMessage } from "../api/chat";
import type { Message } from "../api/types";
import { useRealtimeEvents } from "./useRealtime";

export type ChatMessage = Message & { pending?: boolean; failed?: boolean };

/**
 * Message list of one conversation: first page from REST, live updates from the
 * realtime socket, optimistic sending, and read receipts.
 */
export function useChat(conversationId: string, myId: string, active: boolean) {
  const queryClient = useQueryClient();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sendError, setSendError] = useState<string | null>(null);
  const readTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const markRead = useCallback(() => {
    clearTimeout(readTimer.current);
    readTimer.current = setTimeout(() => {
      markConversationRead(conversationId)
        .then((result) => {
          if (result.read > 0) {
            void queryClient.invalidateQueries({ queryKey: ["conversations"] });
            void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          }
        })
        .catch(() => undefined);
    }, 250);
  }, [conversationId, queryClient]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const page = await fetchMessages(conversationId);
      setMessages(page.messages);
      setNextCursor(page.nextCursor);
      markRead();
    } catch (err) {
      setError(errorMessage(err, "Impossible de charger la conversation."));
    } finally {
      setLoading(false);
    }
  }, [conversationId, markRead]);

  useEffect(() => {
    void load();
    return () => clearTimeout(readTimer.current);
  }, [load]);

  useRealtimeEvents((event) => {
    if (event.type === "message.new" && event.data.conversationId === conversationId) {
      const incoming = event.data;
      setMessages((current) => {
        if (current.some((message) => message.id === incoming.id)) {
          return current;
        }
        // Replace our optimistic copy of a message we sent from this device.
        const pendingIndex = current.findIndex(
          (message) => message.pending && message.content === incoming.content && incoming.senderId === myId
        );
        if (pendingIndex >= 0) {
          const copy = [...current];
          copy[pendingIndex] = incoming;
          return copy;
        }
        return [...current, incoming];
      });
      if (incoming.senderId !== myId && active) {
        markRead();
      }
    }
    if (event.type === "message.read" && event.data.conversationId === conversationId) {
      const readAt = event.data.readAt;
      setMessages((current) => current.map((message) => (message.senderId === myId && !message.readAt ? { ...message, readAt } : message)));
    }
    if (event.type === "ready") {
      void load();
    }
  });

  useEffect(() => {
    if (active) {
      markRead();
    }
  }, [active, markRead]);

  const loadMore = useCallback(async () => {
    if (!nextCursor || loadingMore) {
      return;
    }
    setLoadingMore(true);
    try {
      const page = await fetchMessages(conversationId, nextCursor);
      setMessages((current) => [...page.messages.filter((m) => !current.some((c) => c.id === m.id)), ...current]);
      setNextCursor(page.nextCursor);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingMore(false);
    }
  }, [conversationId, loadingMore, nextCursor]);

  const send = useCallback(
    async (text: string) => {
      const content = text.trim();
      if (!content) {
        return;
      }
      const temporaryId = `pending-${Date.now()}-${Math.random().toString(36).slice(2)}`;
      setSendError(null);
      setMessages((current) => [
        ...current,
        { id: temporaryId, conversationId, senderId: myId, content, createdAt: new Date().toISOString(), pending: true }
      ]);
      try {
        const saved = await sendMessage(conversationId, content);
        setMessages((current) => {
          // The realtime echo may already have replaced the optimistic copy.
          const withoutPending = current.filter((message) => message.id !== temporaryId);
          return withoutPending.some((message) => message.id === saved.id) ? withoutPending : [...withoutPending, saved];
        });
        void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      } catch (err) {
        setMessages((current) => current.filter((message) => message.id !== temporaryId));
        setSendError(errorMessage(err, "Message non envoyé."));
        throw err;
      }
    },
    [conversationId, myId, queryClient]
  );

  return { messages, loading, loadingMore, hasMore: nextCursor !== undefined, error, sendError, loadMore, send, reload: load };
}
