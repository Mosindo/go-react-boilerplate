import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import type { Conversation, Message, MessagesPage, MyProfile } from "../api/types";
import { keys } from "./queryClient";

export function useMyProfile() {
  return useQuery({ queryKey: keys.myProfile, queryFn: profileApi.mine });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: Infinity });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: keys.profile(userId), queryFn: () => profileApi.view(userId) });
}

export function useConversations() {
  return useQuery({ queryKey: keys.conversations, queryFn: () => chatApi.conversations(), refetchInterval: 60_000 });
}

export function useMessages(matchId: string) {
  return useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => chatApi.messages(matchId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => (last.hasMore ? last.messages[last.messages.length - 1]?.id : undefined)
  });
}

export function useNotifications() {
  return useQuery({ queryKey: keys.notifications, queryFn: () => notificationsApi.list(), refetchInterval: 120_000 });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: safetyApi.blocks });
}

/** Writes a fresh profile payload into the cache after a mutation. */
export function useSetMyProfile() {
  const qc = useQueryClient();
  return (next: MyProfile | ((prev: MyProfile) => MyProfile)) => {
    qc.setQueryData<MyProfile>(keys.myProfile, (prev) => (typeof next === "function" ? (prev ? next(prev) : prev) : next));
  };
}

type MessagesCache = InfiniteData<MessagesPage, string | undefined>;

/** Inserts a message at the top of the newest page if it is not cached yet. */
export function insertMessage(cache: MessagesCache | undefined, message: Message): MessagesCache | undefined {
  if (!cache || cache.pages.length === 0) return cache;
  if (cache.pages.some((p) => p.messages.some((m) => m.id === message.id))) return cache;
  const [first, ...rest] = cache.pages;
  return { ...cache, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

export function setOtherReadAt(cache: MessagesCache | undefined, readAt: string): MessagesCache | undefined {
  if (!cache) return cache;
  return { ...cache, pages: cache.pages.map((p, i) => (i === 0 ? { ...p, otherLastReadAt: readAt } : p)) };
}

export function useSendMessage(matchId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => chatApi.send(matchId, body),
    onSuccess: (message) => {
      qc.setQueryData<MessagesCache>(keys.messages(matchId), (cache) => insertMessage(cache, message));
      qc.setQueryData<Conversation[]>(keys.conversations, (list) => bumpConversation(list, message));
    }
  });
}

/** Moves a conversation to the top with its new last message. */
export function bumpConversation(list: Conversation[] | undefined, message: Message, unreadDelta = 0): Conversation[] | undefined {
  if (!list) return list;
  const index = list.findIndex((c) => c.matchId === message.matchId);
  if (index < 0) return list;
  const updated = { ...list[index], lastMessage: message, unreadCount: list[index].unreadCount + unreadDelta };
  return [updated, ...list.slice(0, index), ...list.slice(index + 1)];
}
