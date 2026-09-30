import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api/client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
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

export const queryKeys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  publicProfile: (userId: string) => ["profiles", userId] as const,
  conversations: ["conversations"] as const,
  conversation: (id: string) => ["conversation", id] as const,
  messages: (id: string) => ["messages", id] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const
};
