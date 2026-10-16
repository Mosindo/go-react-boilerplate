import { useQuery } from "@tanstack/react-query";
import {
  getMyProfile,
  getProfile,
  getUnreadCount,
  listBlocked,
  listConversations,
  listInterests,
  listLikesReceived,
  listMatches,
  listNotifications
} from "../api/platform";

export const queryKeys = {
  me: ["me", "profile"] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  unread: ["conversations", "unread"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  matches: ["matches"] as const,
  likes: ["likes"] as const,
  notifications: ["notifications"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  blocked: ["blocked"] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.me, queryFn: getMyProfile, staleTime: 60_000 });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });
}

export function useConversations(realtimeConnected: boolean) {
  return useQuery({
    queryKey: queryKeys.conversations,
    queryFn: () => listConversations(),
    // Without the socket, fall back to light polling so chats still feel alive.
    refetchInterval: realtimeConnected ? false : 20_000
  });
}

export function useUnreadMessages(realtimeConnected: boolean) {
  return useQuery({
    queryKey: queryKeys.unread,
    queryFn: getUnreadCount,
    refetchInterval: realtimeConnected ? false : 20_000
  });
}

export function useMatches() {
  return useQuery({ queryKey: queryKeys.matches, queryFn: () => listMatches() });
}

export function useLikesReceived() {
  return useQuery({ queryKey: queryKeys.likes, queryFn: () => listLikesReceived() });
}

export function useNotifications(realtimeConnected: boolean) {
  return useQuery({
    queryKey: queryKeys.notifications,
    queryFn: () => listNotifications(),
    refetchInterval: realtimeConnected ? false : 30_000
  });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => getProfile(userId), staleTime: 60_000 });
}

export function useBlockedUsers() {
  return useQuery({ queryKey: queryKeys.blocked, queryFn: listBlocked });
}
