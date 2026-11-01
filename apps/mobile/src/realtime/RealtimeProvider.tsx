import { useEffect, type ReactNode } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import { authApi } from "../api";
import { getTokens } from "../api/tokenStore";
import type { MatchEvent, Message, MessagePage } from "../api/types";
import { WS_URL } from "../config";
import { upsertMessage } from "../lib/messages";
import { keys } from "../state/queries";
import { RealtimeClient } from "./socket";

async function freshAccessToken(): Promise<string | null> {
  try {
    await authApi.me(); // declenche un refresh si l'access token a expire
  } catch {
    // reseau indisponible: on tente quand meme avec le token courant
  }
  return (await getTokens())?.accessToken ?? null;
}

/** Ouvre la connexion temps reel tant que l'utilisateur est connecte et synchronise le cache. */
export function RealtimeProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const navigation = useNavigation();

  useEffect(() => {
    const client = new RealtimeClient(WS_URL, freshAccessToken);

    const unsubscribe = client.subscribe((event) => {
      switch (event.type) {
        case "message": {
          const message = event.payload as Message;
          queryClient.setQueryData<InfiniteData<MessagePage>>(keys.messages(message.conversationId), (data) => {
            if (!data || data.pages.length === 0) {
              return data;
            }
            const [first, ...rest] = data.pages;
            return { ...data, pages: [{ ...first, items: upsertMessage(first.items, message) }, ...rest] };
          });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        }
        case "read":
          void queryClient.invalidateQueries({ queryKey: ["messages"] });
          break;
        case "match": {
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          navigation.navigate("MatchModal", event.payload as MatchEvent);
          break;
        }
        case "unmatch":
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
        default:
          break;
      }
    });

    client.start();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        client.reconnectNow();
        void queryClient.invalidateQueries({ queryKey: keys.conversations });
      }
    });

    return () => {
      appState.remove();
      unsubscribe();
      client.stop();
    };
  }, [queryClient, navigation]);

  return <>{children}</>;
}
