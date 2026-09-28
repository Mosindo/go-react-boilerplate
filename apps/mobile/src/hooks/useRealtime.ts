import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, toWebSocketUrl } from "../api/config";
import { endpoints } from "../api/endpoints";
import { getValidAccessToken } from "../api/http";
import { RealtimeClient } from "../api/realtime";
import { queryKeys } from "../api/queryKeys";
import { showToast } from "../shared/feedback";
import { getOpenConversation } from "./openConversation";
import { applyRealtimeEvent } from "./realtimeHandler";

/** Keeps the WebSocket alive while signed in and the app is in the foreground. Renders nothing. */
export function RealtimeBridge({ myUserId }: { myUserId: string }) {
  const queryClient = useQueryClient();

  useEffect(() => {
    const refetchMissed = () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      void queryClient.invalidateQueries({ queryKey: ["messages"] });
    };
    const client = new RealtimeClient({
      url: toWebSocketUrl(API_BASE_URL, endpoints.ws),
      getToken: getValidAccessToken,
      onEvent: (event) =>
        applyRealtimeEvent(queryClient, event, {
          myUserId,
          openConversationId: getOpenConversation(),
          notify: (message) => showToast(message, "success")
        }),
      onReconnected: refetchMissed
    });

    if (AppState.currentState === "active") {
      client.start();
    }
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        client.start();
      } else {
        client.stop();
      }
    });
    return () => {
      subscription.remove();
      client.stop();
    };
  }, [myUserId, queryClient]);

  return null;
}
