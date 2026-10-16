import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { getProfile } from "../api/platform";
import { RealtimeConnection, type RealtimeStatus } from "../api/realtime";
import type { Message, PublicProfile, RealtimeEvent } from "../api/types";
import { queryKeys } from "./queries";
import { useAuth } from "./useAuth";

export type NewMatch = { matchId: string; profile: PublicProfile };

type RealtimeContextValue = {
  connected: boolean;
  /** Match celebration to show (from the live event or a swipe response). */
  newMatch: NewMatch | null;
  announceMatch: (match: NewMatch) => void;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

function appendMessage(existing: { messages: Message[]; hasMore: boolean } | undefined, message: Message) {
  if (!existing) {
    return existing;
  }
  if (existing.messages.some((m) => m.id === message.id)) {
    return existing;
  }
  return { ...existing, messages: [...existing.messages, message] };
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [newMatch, setNewMatch] = useState<NewMatch | null>(null);
  const announced = useRef(new Set<string>());

  const announceMatch = useCallback((match: NewMatch) => {
    if (announced.current.has(match.matchId)) {
      return;
    }
    announced.current.add(match.matchId);
    setNewMatch(match);
  }, []);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          const message = event.data;
          queryClient.setQueryData<{ messages: Message[]; hasMore: boolean }>(queryKeys.messages(message.matchId), (existing) =>
            appendMessage(existing, message)
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.unread });
          break;
        }
        case "message.read":
          void queryClient.invalidateQueries({ queryKey: queryKeys.messages(event.data.matchId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "match.new": {
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
          getProfile(event.data.userId)
            .then((profile) => announceMatch({ matchId: event.data.matchId, profile }))
            .catch(() => undefined);
          break;
        }
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
      }
    },
    [announceMatch, queryClient]
  );

  useEffect(() => {
    if (status !== "signedIn") {
      return;
    }
    const conn = new RealtimeConnection({
      onEvent: handleEvent,
      onStatus: (s: RealtimeStatus) => {
        setConnected(s === "open");
        if (s === "open") {
          // Events may have been missed while offline: resync once per (re)connection.
          void queryClient.invalidateQueries();
        }
      }
    });
    const announcedMatches = announced.current;
    conn.start();

    const subscription = AppState.addEventListener("change", (next) => {
      if (next === "active") {
        conn.reconnectNow();
      }
    });
    return () => {
      subscription.remove();
      conn.stop();
      announcedMatches.clear();
      setNewMatch(null);
    };
  }, [handleEvent, queryClient, status]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      connected,
      newMatch,
      announceMatch,
      dismissMatch: () => setNewMatch(null)
    }),
    [announceMatch, connected, newMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const context = useContext(RealtimeContext);
  if (!context) {
    throw new Error("useRealtime must be used within RealtimeProvider");
  }
  return context;
}
