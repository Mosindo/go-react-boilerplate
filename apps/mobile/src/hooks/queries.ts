import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient
} from "@tanstack/react-query";
import { chatApi, discoveryApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import { mergeFeed } from "../utils/feed";
import type {
  Card,
  ConversationsPage,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  SwipeDirection
} from "../api/types";

export const keys = {
  myProfile: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  profile: (userId: string) => ["profile", userId] as const,
  conversations: ["conversations"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: keys.myProfile, queryFn: profileApi.mine });
}

/** Mutations returning the updated profile write it straight into the cache. */
export function useProfileMutation<TVars>(fn: (vars: TVars) => Promise<MyProfile>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (profile) => {
      qc.setQueryData(keys.myProfile, profile);
      void qc.invalidateQueries({ queryKey: keys.discover });
    }
  });
}

export function useInterestCatalog() {
  return useQuery({
    queryKey: keys.interests,
    queryFn: async () => (await profileApi.interests()).interests,
    staleTime: 24 * 60 * 60 * 1000
  });
}

export function useDiscover(enabled: boolean) {
  const qc = useQueryClient();
  return useQuery({
    queryKey: keys.discover,
    queryFn: async () => mergeFeed(qc.getQueryData<Card[]>(keys.discover), (await discoveryApi.feed(10)).profiles),
    enabled,
    staleTime: 60_000
  });
}

export function useSwipe() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ card, direction }: { card: Card; direction: SwipeDirection }) =>
      discoveryApi.swipe(card.userId, direction),
    // Optimistically drop the card so the next one appears instantly.
    onMutate: async ({ card }) => {
      await qc.cancelQueries({ queryKey: keys.discover });
      const previous = qc.getQueryData<Card[]>(keys.discover);
      qc.setQueryData<Card[]>(keys.discover, (list) => (list ?? []).filter((c) => c.userId !== card.userId));
      return { previous };
    },
    onError: (_error, _vars, ctx) => {
      if (ctx?.previous) {
        qc.setQueryData(keys.discover, ctx.previous);
      }
    },
    onSuccess: (result) => {
      if (result.matched) {
        void qc.invalidateQueries({ queryKey: keys.conversations });
      }
      const remaining = qc.getQueryData<Card[]>(keys.discover) ?? [];
      if (remaining.length < 3) {
        void qc.invalidateQueries({ queryKey: keys.discover });
      }
    }
  });
}

export function useProfileCard(userId: string, initial?: Card) {
  return useQuery({
    queryKey: keys.profile(userId),
    queryFn: () => profileApi.view(userId),
    initialData: initial
  });
}

export function useConversations(pollWhenOffline: boolean) {
  return useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam || undefined),
    initialPageParam: "",
    getNextPageParam: (last: ConversationsPage) => last.nextCursor || undefined,
    refetchInterval: pollWhenOffline ? 15_000 : false
  });
}

export function useMessages(matchId: string, pollWhenOffline: boolean) {
  return useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => chatApi.messages(matchId, pageParam || undefined),
    initialPageParam: "",
    getNextPageParam: (last: MessagesPage) => last.nextCursor || undefined,
    refetchInterval: pollWhenOffline ? 8_000 : false
  });
}

/** Inserts a message into the cached first page if it is not there yet. */
export function addMessageToCache(qc: QueryClient, message: Message) {
  qc.setQueryData<InfiniteData<MessagesPage>>(keys.messages(message.matchId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
      return data;
    }
    const [first, ...rest] = data.pages;
    const withoutPending = first.messages.filter(
      (m) => !(m.id.startsWith("pending-") && m.senderId === message.senderId && m.body === message.body)
    );
    return { ...data, pages: [{ ...first, messages: [...withoutPending, message] }, ...rest] };
  });
}

export function useSendMessage(matchId: string, myUserId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => chatApi.send(matchId, body),
    onMutate: (body) => {
      const pending: Message = {
        id: `pending-${Date.now()}`,
        matchId,
        senderId: myUserId,
        body: body.trim(),
        createdAt: new Date().toISOString()
      };
      addMessageToCache(qc, pending);
      return { pendingId: pending.id };
    },
    onError: (_e, _body, ctx) => {
      qc.setQueryData<InfiniteData<MessagesPage>>(keys.messages(matchId), (data) =>
        data
          ? {
              ...data,
              pages: data.pages.map((p) => ({ ...p, messages: p.messages.filter((m) => m.id !== ctx?.pendingId) }))
            }
          : data
      );
    },
    onSuccess: (message) => {
      addMessageToCache(qc, message);
      void qc.invalidateQueries({ queryKey: keys.conversations });
    }
  });
}

export function useMarkRead(matchId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => chatApi.markRead(matchId),
    onSuccess: () => {
      qc.setQueryData<InfiniteData<ConversationsPage>>(keys.conversations, (data) =>
        data
          ? {
              ...data,
              pages: data.pages.map((p) => ({
                ...p,
                conversations: p.conversations.map((c) => (c.matchId === matchId ? { ...c, unreadCount: 0 } : c))
              }))
            }
          : data
      );
      void qc.invalidateQueries({ queryKey: keys.notifications });
    }
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: 0,
    getNextPageParam: (last: NotificationsPage, all) =>
      last.hasMore ? all.reduce((n, p) => n + p.notifications.length, 0) : undefined
  });
}

export function useUnreadNotifications(): number {
  const { data } = useNotifications();
  return data?.pages[0]?.unreadCount ?? 0;
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: async () => (await safetyApi.blocks()).blocks });
}
