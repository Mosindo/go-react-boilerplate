import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi, discoveryApi, matchesApi, notificationsApi, profileApi, safetyApi, type ProfilePatch } from "../api/dating";
import type { Message, MessagesPage, MyProfile, Preferences, Settings } from "../api/types";

export const keys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  deck: ["deck"] as const,
  matches: ["matches"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  publicProfile: (userId: string) => ["publicProfile", userId] as const,
  notifications: ["notifications"] as const,
  unread: ["notifications", "unread"] as const,
  blocks: ["blocks"] as const
};

export function useMyProfile(enabled = true) {
  return useQuery({ queryKey: keys.profile, queryFn: profileApi.mine, enabled, staleTime: 60_000 });
}

function useProfileMutation<TVariables>(mutationFn: (variables: TVariables) => Promise<MyProfile>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: (profile) => {
      client.setQueryData(keys.profile, profile);
      void client.invalidateQueries({ queryKey: keys.deck });
    }
  });
}

export function useUpdateProfile() {
  return useProfileMutation((patch: ProfilePatch) => profileApi.update(patch));
}

export function useUpdatePreferences() {
  return useProfileMutation((prefs: Preferences) => profileApi.updatePreferences(prefs));
}

export function useUpdateSettings() {
  return useProfileMutation((settings: Settings) => profileApi.updateSettings(settings));
}

export function useUpdateLocation() {
  return useProfileMutation((loc: { latitude: number; longitude: number; city?: string }) =>
    profileApi.updateLocation(loc.latitude, loc.longitude, loc.city)
  );
}

export function useClearLocation() {
  return useProfileMutation(() => profileApi.clearLocation());
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.publicProfile(userId) });
}

export function useDeck(enabled: boolean) {
  return useQuery({ queryKey: keys.deck, queryFn: () => discoveryApi.deck(10), enabled, staleTime: 5 * 60_000 });
}

export function useMatches() {
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => (last.hasMore ? pages.reduce((n, p) => n + p.matches.length, 0) : undefined)
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[last.messages.length - 1]?.id : undefined),
    staleTime: 0
  });
}

/** Inserts a message at the top of the newest page, deduplicating by id. */
export function insertMessage(data: InfiniteData<MessagesPage, string | undefined> | undefined, message: Message) {
  if (!data || data.pages.length === 0) {
    return data;
  }
  if (data.pages.some((page) => page.messages.some((m) => m.id === message.id))) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

export function useSendMessage(conversationId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (content: string) => chatApi.send(conversationId, content),
    onSuccess: (message) => {
      client.setQueryData<InfiniteData<MessagesPage, string | undefined>>(keys.messages(conversationId), (data) =>
        insertMessage(data, message)
      );
      void client.invalidateQueries({ queryKey: keys.matches });
    }
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => (last.hasMore ? pages.reduce((n, p) => n + p.notifications.length, 0) : undefined)
  });
}

export function useUnreadNotifications(enabled: boolean) {
  return useQuery({ queryKey: keys.unread, queryFn: notificationsApi.unreadCount, enabled, staleTime: 60_000 });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: safetyApi.blocks });
}
