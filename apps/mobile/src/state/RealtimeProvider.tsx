import { useQueryClient } from "@tanstack/react-query";
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState
} from "react";
import { AppState } from "react-native";
import { API_BASE_URL, toWebSocketUrl } from "../config";
import { getTokensSync } from "../api/session";
import type { Match, RealtimeEvent } from "../api/types";
import { appendMessage, markReadBy, type MessagesCache } from "./messageCache";
import { useMe } from "./AuthProvider";

type RealtimeApi = {
  connected: boolean;
  /** Latest match announced by the server, shown in a celebration modal. */
  newMatch: Match | null;
  /**
   * Shows the match modal. The swipe response calls it so the celebration appears
   * at once even if the WebSocket is down; the pushed "match" event does the same.
   */
  announceMatch: (match: Match) => void;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeApi>({
  connected: false,
  newMatch: null,
  announceMatch: () => undefined,
  dismissMatch: () => undefined
});

const MAX_BACKOFF_MS = 30_000;

function parseEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (
      typeof parsed === "object" &&
      parsed !== null &&
      typeof (parsed as { type?: unknown }).type === "string"
    ) {
      return parsed as RealtimeEvent;
    }
  } catch {
    // Ignore malformed frames.
  }
  return null;
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [newMatch, setNewMatch] = useState<Match | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        case "message": {
          const message = event.data;
          queryClient.setQueryData<MessagesCache>(["messages", message.conversationId], (cache) =>
            appendMessage(cache, message)
          );
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          break;
        }
        case "read": {
          const { conversationId, readerId, readAt } = event.data;
          queryClient.setQueryData<MessagesCache>(["messages", conversationId], (cache) =>
            markReadBy(cache, readerId, readAt)
          );
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          break;
        }
        case "match":
          setNewMatch(event.data);
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["discover"] });
          break;
        case "notification":
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          break;
        case "unmatch":
          queryClient.removeQueries({ queryKey: ["messages", event.data.conversationId] });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          break;
        default:
          break;
      }
    },
    [queryClient]
  );

  useEffect(() => {
    if (!API_BASE_URL) {
      return;
    }
    let stopped = false;
    let retries = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      const token = getTokensSync()?.accessToken;
      if (stopped || !token) {
        return;
      }
      const socket = new WebSocket(toWebSocketUrl(API_BASE_URL));
      socketRef.current = socket;

      socket.onopen = () => {
        socket.send(JSON.stringify({ type: "auth", token }));
      };
      socket.onmessage = (frame) => {
        const event = parseEvent(frame.data);
        if (!event) return;
        if (event.type === "ready") {
          retries = 0;
          setConnected(true);
          // Anything that happened while offline is picked up by refetching.
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          void queryClient.invalidateQueries({ queryKey: ["messages"] });
          return;
        }
        handleEvent(event);
      };
      socket.onclose = () => {
        setConnected(false);
        if (socketRef.current === socket) socketRef.current = null;
        if (stopped) return;
        retries += 1;
        timer = setTimeout(connect, Math.min(1000 * 2 ** retries, MAX_BACKOFF_MS));
      };
      socket.onerror = () => {
        socket.close();
      };
    };

    connect();
    const appStateSub = AppState.addEventListener("change", (status) => {
      if (status === "active" && !socketRef.current && !stopped) {
        if (timer) clearTimeout(timer);
        retries = 0;
        connect();
      }
    });

    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      appStateSub.remove();
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [handleEvent, queryClient, me.id]);

  const dismissMatch = useCallback(() => setNewMatch(null), []);
  const announceMatch = useCallback((match: Match) => setNewMatch(match), []);
  const value = useMemo(
    () => ({ connected, newMatch, announceMatch, dismissMatch }),
    [connected, newMatch, announceMatch, dismissMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeApi {
  return useContext(RealtimeContext);
}
