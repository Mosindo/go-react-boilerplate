import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
      retry: (count, error) => {
        // Client errors are final; retry transient failures twice.
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          return false;
        }
        return count < 2;
      }
    },
    mutations: { retry: 0 }
  }
});

export const keys = {
  me: ["me"] as const,
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  conversations: ["conversations"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};
