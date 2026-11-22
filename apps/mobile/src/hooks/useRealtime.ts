import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { connectRealtime, type RealtimeEvent } from "../api/realtime";
import type { Match, Message } from "../api/types";
import { showToast } from "../ui";
import { qk, setOtherLastRead, upsertMessage } from "./queries";

/**
 * Keeps React Query caches in sync with server events while signed in, so
 * lists update without polling. On reconnect or app foreground, caches are
 * revalidated to recover anything missed while offline.
 */
export function useRealtime(currentUserId: string, activeChatUserId: { current: string | null }) {
  const client = useQueryClient();

  useEffect(() => {
    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.created": {
          const message = event.data as Message;
          const other = message.senderUserId === currentUserId ? message.recipientUserId : message.senderUserId;
          upsertMessage(client, other, message);
          void client.invalidateQueries({ queryKey: qk.chats });
          break;
        }
        case "conversation.read": {
          const data = event.data as { userId: string; readAt: string };
          if (data.userId !== currentUserId) {
            setOtherLastRead(client, data.userId, data.readAt);
          }
          void client.invalidateQueries({ queryKey: qk.chats });
          break;
        }
        case "match.created": {
          const match = event.data as Match;
          void client.invalidateQueries({ queryKey: qk.chats });
          void client.invalidateQueries({ queryKey: qk.matches });
          void client.invalidateQueries({ queryKey: qk.likes });
          if (activeChatUserId.current !== match.user.userId) {
            showToast(`Nouveau match avec ${match.user.firstName} ✨`, "success");
          }
          break;
        }
        case "match.removed": {
          const data = event.data as { userId: string };
          void client.invalidateQueries({ queryKey: qk.chats });
          void client.invalidateQueries({ queryKey: qk.matches });
          void client.invalidateQueries({ queryKey: qk.likes });
          client.removeQueries({ queryKey: qk.publicProfile(data.userId) });
          void client.invalidateQueries({ queryKey: qk.messages(data.userId) });
          break;
        }
        case "like.received":
          void client.invalidateQueries({ queryKey: qk.likes });
          break;
        case "notification.created":
          void client.invalidateQueries({ queryKey: qk.notifications });
          break;
        default:
          break;
      }
    };

    const resync = () => {
      void client.invalidateQueries({ queryKey: qk.chats });
      void client.invalidateQueries({ queryKey: qk.notifications });
      void client.invalidateQueries({ queryKey: qk.likes });
      if (activeChatUserId.current) {
        void client.invalidateQueries({ queryKey: qk.messages(activeChatUserId.current) });
      }
    };

    let wasConnected = false;
    const disconnect = connectRealtime({
      onEvent: handle,
      onStatus: (connected) => {
        if (connected && wasConnected === false) {
          resync();
        }
        wasConnected = connected;
      }
    });
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        resync();
      }
    });
    return () => {
      disconnect();
      sub.remove();
    };
  }, [activeChatUserId, client, currentUserId]);
}
