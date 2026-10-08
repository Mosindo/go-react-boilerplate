import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { keys } from "../api/keys";
import * as api from "../api/platform";
import type { Me, Message, MessagesPage, ProfileInput, Preferences, SwipeAction } from "../api/types";
import { useAuth } from "./useAuth";

function useSignedIn(): boolean {
  return useAuth().status === "signedIn";
}

export function useMe() {
  const enabled = useSignedIn();
  return useQuery({ queryKey: keys.me, queryFn: api.getMe, enabled });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: api.listInterests, staleTime: Infinity });
}

export function useProfile(id: string) {
  return useQuery({ queryKey: keys.profile(id), queryFn: () => api.getProfile(id), enabled: Boolean(id) });
}

export function useDiscover(enabled: boolean) {
  return useQuery({ queryKey: keys.discover, queryFn: () => api.getDiscover(10), enabled, staleTime: 0, gcTime: 0 });
}

export function useMatches() {
  const enabled = useSignedIn();
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => api.getMatches(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    enabled
  });
}

export function useConversations() {
  const enabled = useSignedIn();
  return useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => api.getConversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    enabled
  });
}

export function useUnread() {
  const enabled = useSignedIn();
  return useQuery({ queryKey: keys.unread, queryFn: api.getUnread, enabled });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => api.getMessages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 0
  });
}

export function useNotifications() {
  const enabled = useSignedIn();
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => api.getNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
    enabled
  });
}

export function useNotificationsUnread() {
  const enabled = useSignedIn();
  return useQuery({ queryKey: keys.notificationsUnread, queryFn: api.getNotificationsUnread, enabled });
}

export function useBlocked() {
  return useQuery({ queryKey: keys.blocked, queryFn: api.getBlocked, staleTime: 0 });
}

/** Writes the fresh `Me` returned by profile mutations straight into the cache. */
export function useSaveProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: ProfileInput) => api.saveProfile(input),
    onSuccess: (me: Me) => client.setQueryData(keys.me, me)
  });
}

export function useSavePreferences() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: Preferences) => api.savePreferences(input),
    onSuccess: (me: Me) => {
      client.setQueryData(keys.me, me);
      void client.invalidateQueries({ queryKey: keys.discover });
    }
  });
}

export function useSwipe() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, action }: { userId: string; action: SwipeAction }) => api.swipe(userId, action),
    onSuccess: (result) => {
      if (result.matched && !result.alreadyProcessed) {
        void client.invalidateQueries({ queryKey: keys.matches });
        void client.invalidateQueries({ queryKey: keys.notifications });
        void client.invalidateQueries({ queryKey: keys.notificationsUnread });
      }
    }
  });
}

export function useSendMessage(conversationId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api.sendMessage(conversationId, body),
    onSuccess: (message: Message) => {
      client.setQueryData<InfiniteData<MessagesPage>>(keys.messages(conversationId), (data) => {
        if (!data || data.pages.length === 0) return data;
        if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) return data;
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
      });
      void client.invalidateQueries({ queryKey: keys.conversations });
      void client.invalidateQueries({ queryKey: keys.matches });
    }
  });
}

export function useMarkRead(conversationId: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => api.markRead(conversationId),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.conversations });
      void client.invalidateQueries({ queryKey: keys.unread });
    }
  });
}
