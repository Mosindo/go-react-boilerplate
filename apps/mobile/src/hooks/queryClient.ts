import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
      // Never retry client errors (4xx): they will not fix themselves.
      retry: (failureCount, error) => {
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
          return false;
        }
        return failureCount < 2;
      }
    },
    mutations: {
      retry: 0
    }
  }
});

export const queryKeys = {
  me: ["me"] as const,
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  chatUnread: ["conversations", "unread"] as const,
  notifications: ["notifications"] as const,
  notificationsUnread: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};
