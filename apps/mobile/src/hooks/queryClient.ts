import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        // Client errors (4xx) are not transient: retrying them only wastes requests.
        if (error instanceof ApiError && error.status >= 400 && error.status < 500) return false;
        return failureCount < 2;
      }
    },
    mutations: { retry: 0 }
  }
});

export const queryKeys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  publicProfile: (userId: string) => ["public-profile", userId] as const,
  matches: ["matches"] as const,
  likes: ["likes"] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  notifications: ["notifications"] as const,
  unreadConversations: ["unread", "conversations"] as const,
  unreadNotifications: ["unread", "notifications"] as const,
  blocked: ["blocked"] as const
};
