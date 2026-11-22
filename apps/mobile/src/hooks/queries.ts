import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import * as dating from "../api/dating";
import type { ChatsResponse, Message, MessagesPage, OwnProfile, Photo } from "../api/types";

export const qk = {
  profile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discovery: ["discovery"] as const,
  likes: ["likes"] as const,
  matches: ["matches"] as const,
  chats: ["chats"] as const,
  messages: (userId: string) => ["messages", userId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  publicProfile: (userId: string) => ["profile", userId] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: qk.profile, queryFn: dating.getMyProfile });
}

export function useInterests() {
  return useQuery({ queryKey: qk.interests, queryFn: dating.listInterests, staleTime: Infinity });
}

/** Keeps the cached own profile in sync after any profile mutation. */
export function useProfileMutation<TArgs>(fn: (args: TArgs) => Promise<OwnProfile>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (profile) => {
      client.setQueryData(qk.profile, profile);
      void client.invalidateQueries({ queryKey: qk.discovery });
    }
  });
}

export function usePhotosMutation<TArgs>(fn: (args: TArgs) => Promise<Photo[]>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: qk.profile });
    }
  });
}

export function useLikesReceived() {
  return useQuery({ queryKey: qk.likes, queryFn: dating.getLikesReceived });
}

export function useChats() {
  return useQuery({ queryKey: qk.chats, queryFn: dating.getChats, refetchInterval: 60_000 });
}

export function useNotifications() {
  return useQuery({ queryKey: qk.notifications, queryFn: dating.getNotifications });
}

export function useBlocks() {
  return useQuery({ queryKey: qk.blocks, queryFn: dating.getBlocks });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: qk.publicProfile(userId), queryFn: () => dating.getProfile(userId), retry: false });
}

export function useMessages(userId: string) {
  return useInfiniteQuery({
    queryKey: qk.messages(userId),
    queryFn: ({ pageParam }) => dating.getMessages(userId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore && last.messages.length > 0 ? last.messages[0].id : undefined),
    retry: false
  });
}

type MessagesData = InfiniteData<MessagesPage, string | undefined>;

/** Inserts (or replaces) a message in the newest cached page, de-duplicated by id. */
export function upsertMessage(client: QueryClient, otherUserId: string, message: Message, replaceId?: string) {
  client.setQueryData<MessagesData>(qk.messages(otherUserId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    const [newest, ...older] = data.pages;
    const kept = newest.messages.filter((m) => m.id !== message.id && m.id !== replaceId);
    return { ...data, pages: [{ ...newest, messages: [...kept, message] }, ...older] };
  });
}

export function removeMessage(client: QueryClient, otherUserId: string, id: string) {
  client.setQueryData<MessagesData>(qk.messages(otherUserId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    const [newest, ...older] = data.pages;
    return { ...data, pages: [{ ...newest, messages: newest.messages.filter((m) => m.id !== id) }, ...older] };
  });
}

export function setOtherLastRead(client: QueryClient, otherUserId: string, readAt: string) {
  client.setQueryData<MessagesData>(qk.messages(otherUserId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    const [newest, ...older] = data.pages;
    return { ...data, pages: [{ ...newest, otherLastReadAt: readAt }, ...older] };
  });
}

export function clearUnread(client: QueryClient, otherUserId: string) {
  client.setQueryData<ChatsResponse>(qk.chats, (data) => {
    if (!data) {
      return data;
    }
    let cleared = 0;
    const chats = data.chats.map((c) => {
      if (c.user.id !== otherUserId) {
        return c;
      }
      cleared = c.unreadCount;
      return { ...c, unreadCount: 0 };
    });
    return { chats, totalUnread: Math.max(0, data.totalUnread - cleared) };
  });
}
