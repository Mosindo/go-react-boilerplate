import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  getBlocks,
  getConversations,
  getInterests,
  getMatches,
  getMessages,
  getMyProfile,
  getNotifications,
  getProfile
} from "../api/platform";
import { queryKeys, type MessagesData } from "../realtime/cache";

const MINUTE = 60_000;

export const useMyProfile = (enabled = true) =>
  useQuery({ queryKey: queryKeys.me, queryFn: ({ signal }) => getMyProfile(signal), enabled, staleTime: MINUTE });

export const useInterests = () => useQuery({ queryKey: queryKeys.interests, queryFn: getInterests, staleTime: 60 * MINUTE });

export const useMatches = () => useQuery({ queryKey: queryKeys.matches, queryFn: ({ signal }) => getMatches(signal), staleTime: 30_000 });

export const useConversations = () =>
  useQuery({ queryKey: queryKeys.conversations, queryFn: ({ signal }) => getConversations(signal), staleTime: 30_000 });

export const useNotifications = () =>
  useQuery({ queryKey: queryKeys.notifications, queryFn: ({ signal }) => getNotifications(signal), staleTime: 30_000 });

export const useBlocks = () => useQuery({ queryKey: queryKeys.blocks, queryFn: getBlocks });

export const useProfileCard = (userId: string) =>
  useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => getProfile(userId), staleTime: 5 * MINUTE });

/** Chat history, newest page first; `fetchNextPage` loads older messages. */
export const useMessages = (conversationId: string) =>
  useInfiniteQuery<MessagesData["pages"][number], Error, MessagesData, ReturnType<typeof queryKeys.messages>, string | undefined>({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam, signal }) => getMessages(conversationId, pageParam, signal),
    initialPageParam: undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    staleTime: 30_000
  });

/** Total unread messages across conversations (tab badge). */
export function useUnreadMessages(): number {
  const { data } = useConversations();
  return data?.reduce((sum, c) => sum + c.unreadCount, 0) ?? 0;
}
