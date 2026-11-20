import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
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

/** Centralised query keys so cache updates stay consistent. */
export const keys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  likes: ["likes"] as const,
  conversations: (filter: string) => ["conversations", filter] as const,
  conversationsRoot: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  profileCard: (userId: string) => ["profile-card", userId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const
};
