import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 15_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

export const queryKeys = {
  ownProfile: ["profile", "own"] as const,
  interests: ["interests"] as const,
  publicProfile: (id: string) => ["profile", "public", id] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const
};
