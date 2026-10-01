import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./client";

export const queryKeys = {
  me: ["me"] as const,
  profile: ["profile", "own"] as const,
  preferences: ["profile", "preferences"] as const,
  publicProfile: (userId: string) => ["profile", "public", userId] as const,
  interests: ["interests"] as const,
  photos: ["photos"] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 20_000,
        refetchOnWindowFocus: false,
        // client errors (4xx) will not fix themselves: do not retry them
        retry: (failureCount, error) => !(error instanceof ApiError && error.status >= 400 && error.status < 500) && failureCount < 2
      },
      mutations: { retry: 0 }
    }
  });
}
