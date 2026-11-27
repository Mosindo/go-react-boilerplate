import { AppState, Platform } from "react-native";
import { focusManager, QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      retry: (failureCount, error) => {
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          return false;
        }
        return failureCount < 2;
      }
    },
    mutations: { retry: 0 }
  }
});

// Refetch stale queries when the app returns to the foreground.
if (Platform.OS !== "web") {
  AppState.addEventListener("change", (status) => {
    focusManager.setFocused(status === "active");
  });
}

export const keys = {
  myProfile: ["myProfile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  matches: ["matches"] as const,
  match: (id: string) => ["match", id] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  profile: (userId: string) => ["profile", userId] as const,
  notifications: ["notifications"] as const,
  unread: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};
