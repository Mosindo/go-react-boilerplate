import { useQuery } from "@tanstack/react-query";
import { chatApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import { ApiError } from "../api/client";

export const keys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const,
  publicProfile: (userId: string) => ["publicProfile", userId] as const
};

/** Resolves to null while the account has no profile yet (HTTP 404). */
export function useProfile() {
  return useQuery({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await profileApi.get();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    }
  });
}

export const useInterests = () =>
  useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: Infinity });
export const useConversations = () => useQuery({ queryKey: keys.conversations, queryFn: chatApi.conversations });
export const useNotifications = () => useQuery({ queryKey: keys.notifications, queryFn: notificationsApi.list });
export const useBlocked = () => useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
export const usePublicProfile = (userId: string) =>
  useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.publicProfile(userId), retry: false });

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
