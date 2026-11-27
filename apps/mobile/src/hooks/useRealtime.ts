import { useEffect, useSyncExternalStore } from "react";
import { AppState } from "react-native";
import type { InfiniteData } from "@tanstack/react-query";
import { matchesApi } from "../api/endpoints";
import { realtime, type RealtimeEvent } from "../api/realtime";
import type { ChatMessage, MatchSummary, MessagesPage } from "../api/types";
import { toast } from "../ui";
import { keys, queryClient } from "./queryClient";

// --- Active chat registry (to auto-mark read and react to unmatches) --------

let activeMatchId: string | null = null;
const removedListeners = new Set<(matchId: string) => void>();
const announcedMatches = new Set<string>();

export function setActiveChat(matchId: string | null) {
  activeMatchId = matchId;
}

export function onMatchRemoved(listener: (matchId: string) => void): () => void {
  removedListeners.add(listener);
  return () => removedListeners.delete(listener);
}

const likesInFlight = new Set<string>();

/** Avoids a "new match" toast when the UI already celebrates it. */
export function markMatchAnnounced(matchId: string) {
  announcedMatches.add(matchId);
}

/**
 * Marks a like being sent: the match it may create is celebrated by the
 * swipe screen, even if the realtime event arrives before the HTTP reply.
 */
export function markLikeInFlight(userId: string, inFlight: boolean) {
  if (inFlight) {
    likesInFlight.add(userId);
  } else {
    setTimeout(() => likesInFlight.delete(userId), 3000);
  }
}

// --- Cache helpers ----------------------------------------------------------

type MessagesData = InfiniteData<MessagesPage, string | undefined>;

/** Inserts or replaces a message (matching by id or clientId), newest first. */
export function upsertMessage(matchId: string, message: ChatMessage) {
  queryClient.setQueryData<MessagesData>(keys.messages(matchId), (data) => {
    if (!data) {
      return data;
    }
    let replaced = false;
    const pages = data.pages.map((page) => ({
      ...page,
      messages: page.messages.map((m) => {
        if (m.id === message.id || (message.clientId && m.clientId === message.clientId && m.senderUserId === message.senderUserId)) {
          replaced = true;
          return { ...message, pending: false, failed: false };
        }
        return m;
      })
    }));
    if (!replaced && pages.length > 0) {
      pages[0] = { ...pages[0], messages: [message, ...pages[0].messages] };
    }
    return { ...data, pages };
  });
}

function applyReadReceipt(matchId: string, readAt: string) {
  const at = new Date(readAt).getTime();
  queryClient.setQueryData<MessagesData>(keys.messages(matchId), (data) => {
    if (!data) {
      return data;
    }
    return {
      ...data,
      pages: data.pages.map((page, index) => ({
        ...page,
        otherLastReadAt: index === 0 ? readAt : page.otherLastReadAt,
        messages: page.messages.map((m) => (!m.pending && new Date(m.createdAt).getTime() <= at ? { ...m, isRead: true } : m))
      }))
    };
  });
}

function handleEvent(event: RealtimeEvent, myId: string) {
  switch (event.type) {
    case "ready":
      // (Re)connected: resynchronise anything missed while offline.
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
      if (activeMatchId) {
        void queryClient.invalidateQueries({ queryKey: keys.messages(activeMatchId) });
      }
      break;
    case "message.new": {
      const message = event.payload as ChatMessage;
      upsertMessage(message.matchId, message);
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      if (message.senderUserId !== myId && activeMatchId === message.matchId) {
        void matchesApi.markRead(message.matchId).catch(() => undefined);
      }
      break;
    }
    case "message.read": {
      const receipt = event.payload as { matchId: string; userId: string; readAt: string };
      if (receipt.userId !== myId) {
        applyReadReceipt(receipt.matchId, receipt.readAt);
      } else {
        void queryClient.invalidateQueries({ queryKey: keys.matches });
      }
      break;
    }
    case "match.new": {
      const match = event.payload as MatchSummary;
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      if (!announcedMatches.has(match.id) && !likesInFlight.has(match.user.userId)) {
        announcedMatches.add(match.id);
        toast(`Nouveau match avec ${match.user.firstName} !`, "success");
      }
      break;
    }
    case "match.removed": {
      const { matchId } = event.payload as { matchId: string };
      queryClient.removeQueries({ queryKey: keys.messages(matchId) });
      queryClient.removeQueries({ queryKey: keys.match(matchId) });
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      removedListeners.forEach((l) => l(matchId));
      break;
    }
    case "notification.new":
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
      break;
  }
}

/** Connects the WebSocket while signed in and keeps caches in sync. */
export function useRealtimeSync(myId: string) {
  useEffect(() => {
    if (!myId) {
      return;
    }
    const unsubscribe = realtime.subscribe((event) => handleEvent(event, myId));
    realtime.start();
    const appState = AppState.addEventListener("change", (status) => {
      if (status === "active" && !realtime.connected) {
        realtime.reconnect();
      }
    });
    return () => {
      unsubscribe();
      appState.remove();
      realtime.stop();
    };
  }, [myId]);
}

export function useRealtimeConnected(): boolean {
  return useSyncExternalStore(
    (cb) => realtime.onStatus(cb),
    () => realtime.connected,
    () => realtime.connected
  );
}
