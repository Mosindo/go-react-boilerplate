import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { AppState } from "react-native";
import { api } from "../api/client";
import { API_BASE_URL, toWebSocketUrl } from "../api/config";
import { RealtimeClient } from "../api/realtime";
import { applyRealtimeEvent, invalidateAfterReconnect } from "../api/realtimeCache";
import { getActiveConversation } from "../store/activeConversation";
import { tokenStore } from "../store/tokenStore";
import { toast } from "../ui";
import { useCurrentUserId } from "./useAuth";

/** Keeps one WebSocket open while the user is signed in and routes events into the query cache. */
export function useRealtimeConnection(): void {
  const client = useQueryClient();
  const myUserId = useCurrentUserId();

  useEffect(() => {
    const realtime = new RealtimeClient({
      url: toWebSocketUrl(API_BASE_URL),
      getToken: tokenStore.getAccessToken,
      refreshSession: () => api.refreshSession(tokenStore.getAccessToken()),
      onEvent: (event) => {
        applyRealtimeEvent(client, event, { myUserId, getActiveConversationId: getActiveConversation });
        if (event.type === "match.new") toast.info(`It's a match with ${event.data.user.displayName}!`);
      },
      onReconnect: () => invalidateAfterReconnect(client)
    });
    realtime.start();
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") realtime.reconnectNow();
    });
    return () => {
      subscription.remove();
      realtime.stop();
    };
  }, [client, myUserId]);
}
