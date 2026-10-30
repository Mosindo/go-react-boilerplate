import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { eventsApi } from "../api/endpoints";
import { ApiError } from "../api/client";
import type { RealtimeEvent } from "../api/types";
import { useToast } from "../ui";
import { keys } from "./queries";

const RETRY_DELAY_MS = 3_000;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Keeps the app in sync through the API's long-poll endpoint. Events only say
 * that something changed; the affected queries are refetched through the
 * normal authorized endpoints.
 */
export function useRealtime(enabled: boolean): void {
  const queryClient = useQueryClient();
  const toast = useToast();

  useEffect(() => {
    if (!enabled) return undefined;
    const controller = new AbortController();

    const apply = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message":
        case "read":
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          if (event.conversationId) {
            void queryClient.invalidateQueries({ queryKey: keys.messages(event.conversationId) });
          }
          if (event.type === "message") void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "match":
          toast.show("C'est un match !");
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "notification":
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "unmatch":
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
      }
    };

    (async () => {
      let cursor: number | null = null;
      while (!controller.signal.aborted) {
        if (AppState.currentState !== "active") {
          await sleep(1_000);
          continue;
        }
        try {
          if (cursor === null) {
            cursor = (await eventsApi.cursor()).cursor;
            // Anything that happened while the app was closed is picked up by a refetch.
            void queryClient.invalidateQueries();
            continue;
          }
          const result = await eventsApi.poll(cursor, controller.signal);
          cursor = result.cursor;
          result.events.forEach(apply);
        } catch (error) {
          if (controller.signal.aborted) return;
          if (error instanceof ApiError && error.status === 401) return;
          await sleep(RETRY_DELAY_MS);
        }
      }
    })();

    return () => controller.abort();
  }, [enabled, queryClient, toast]);
}
