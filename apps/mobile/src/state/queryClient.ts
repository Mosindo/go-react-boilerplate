import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        // Client errors are definitive; only retry transient failures.
        if (error instanceof ApiError && error.status < 500) {
          return false;
        }
        return failureCount < 2;
      }
    },
    mutations: { retry: 0 }
  }
});

export const queryKeys = {
  me: ["me"] as const,
  myProfile: ["profile", "me"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  messages: (userId: string) => ["messages", userId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const
};
