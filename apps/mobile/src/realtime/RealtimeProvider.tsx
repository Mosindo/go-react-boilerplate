import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, http, toWebSocketUrl } from "../api";
import { useSession } from "../session/SessionProvider";
import { activeChat } from "./activeChat";
import { applyServerEvent, parseServerEvent, refetchAfterReconnect } from "./events";
import { RealtimeClient, type SocketStatus } from "./socket";

const RealtimeStatusContext = createContext<SocketStatus>("idle");

/** Keeps one WebSocket open while signed in and funnels events into the React Query cache. */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const { userId } = useSession();
  const [status, setStatus] = useState<SocketStatus>("idle");
  const clientRef = useRef<RealtimeClient | null>(null);

  useEffect(() => {
    if (!userId) return;
    const client = new RealtimeClient({
      url: toWebSocketUrl(API_BASE_URL),
      getToken: () => http.getFreshAccessToken(),
      onStatus: setStatus,
      onReconnect: () => refetchAfterReconnect(queryClient),
      onMessage: (raw) => {
        const event = parseServerEvent(raw);
        if (event) {
          applyServerEvent(queryClient, event, {
            myUserId: userId,
            activeMatchId: activeChat.get()
          });
        }
      }
    });
    clientRef.current = client;
    client.start();
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") client.reconnectNow();
    });
    return () => {
      subscription.remove();
      client.stop();
      clientRef.current = null;
    };
  }, [queryClient, userId]);

  return <RealtimeStatusContext.Provider value={status}>{children}</RealtimeStatusContext.Provider>;
}

export function useRealtimeStatus(): SocketStatus {
  return useContext(RealtimeStatusContext);
}
