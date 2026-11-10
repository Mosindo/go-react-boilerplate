import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { matchesApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import type { Match, MeProfile, Message, Page } from "../api/types";

export const keys = {
  me: ["me-profile"] as const,
  interests: ["interests"] as const,
  matches: ["matches"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  unread: ["notifications", "unread"] as const,
  blocked: ["blocked"] as const,
  profile: (userId: string) => ["profile", userId] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: keys.me, queryFn: profileApi.me, staleTime: 60_000 });
}

export function useSetMyProfile() {
  const qc = useQueryClient();
  return (updater: (prev: MeProfile) => MeProfile) => {
    qc.setQueryData<MeProfile>(keys.me, (prev) => (prev ? updater(prev) : prev));
  };
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 24 * 3600_000 });
}

export function useMatches() {
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useMessages(matchId: string, pollWhileOffline: boolean) {
  return useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => matchesApi.messages(matchId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: pollWhileOffline ? 8_000 : false
  });
}

export function useUnreadCount() {
  return useQuery({ queryKey: keys.unread, queryFn: notificationsApi.unreadCount, staleTime: 30_000 });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useBlocked() {
  return useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
}

export function useProfileCard(userId: string) {
  return useQuery({ queryKey: keys.profile(userId), queryFn: () => profileApi.view(userId) });
}

// --- cache helpers shared by realtime and mutations ---------------------------

type MessagesData = InfiniteData<Page<Message>, string | null>;
type MatchesData = InfiniteData<Page<Match>, string | null>;

export function insertMessage(data: MessagesData | undefined, message: Message): MessagesData | undefined {
  if (!data) return data;
  const exists = data.pages.some((p) => p.items.some((m) => m.id === message.id));
  if (exists) return data;
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, items: [message, ...first.items] }, ...rest] };
}

export function markMessagesRead(data: MessagesData | undefined, readerId: string, readAt: string): MessagesData | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      items: p.items.map((m) => (m.senderId !== readerId && !m.readAt ? { ...m, readAt } : m))
    }))
  };
}

export function removeMatch(data: MatchesData | undefined, matchId: string): MatchesData | undefined {
  if (!data) return data;
  return { ...data, pages: data.pages.map((p) => ({ ...p, items: p.items.filter((m) => m.id !== matchId) })) };
}

export function useSendMessage(matchId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => matchesApi.send(matchId, body),
    onSuccess: (message) => {
      qc.setQueryData<MessagesData>(keys.messages(matchId), (d) => insertMessage(d, message));
      void qc.invalidateQueries({ queryKey: keys.matches });
    }
  });
}
