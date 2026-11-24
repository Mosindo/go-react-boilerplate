import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        // Never retry client errors (validation, permissions, not found).
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
        return failureCount < 2;
      }
    },
    mutations: { retry: 0 }
  }
});

export const queryKeys = {
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  card: (userId: string) => ["card", userId] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  unreadNotifications: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};
