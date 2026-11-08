import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { chatApi, matchesApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";

export const keys = {
  profile: ["profile"] as const,
  preferences: ["preferences"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  matches: ["matches"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  unread: ["unread"] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const,
  userProfile: (id: string) => ["userProfile", id] as const
};

export const useMyProfile = () => useQuery({ queryKey: keys.profile, queryFn: profileApi.me });
export const usePreferences = () => useQuery({ queryKey: keys.preferences, queryFn: profileApi.preferences });
export const useInterests = () =>
  useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: Infinity });
export const useBlocked = () => useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
export const useUserProfile = (id: string) =>
  useQuery({ queryKey: keys.userProfile(id), queryFn: () => profileApi.get(id) });
export const useUnreadMessages = () =>
  useQuery({ queryKey: keys.unread, queryFn: chatApi.unreadCount, refetchInterval: 60_000 });

export const useMatches = () =>
  useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });

export const useConversations = () =>
  useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });

export const useMessages = (conversationId: string) =>
  useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });

export const useNotifications = () =>
  useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });
