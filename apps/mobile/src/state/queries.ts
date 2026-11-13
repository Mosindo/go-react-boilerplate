import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { chatApi } from "../api/chat";
import { notificationsApi } from "../api/notifications";
import { profileApi } from "../api/profile";
import type { Conversation, Message, MessagesPage, OwnProfile, UpdateProfileInput } from "../api/types";
import { queryKeys } from "./queryClient";

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.getMine });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

/** Writes the server's returned profile straight into the cache. */
export function useProfileMutation<TVars>(mutationFn: (vars: TVars) => Promise<OwnProfile>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: (profile) => {
      client.setQueryData(queryKeys.myProfile, profile);
    }
  });
}

export function useUpdateProfile() {
  return useProfileMutation((input: UpdateProfileInput) => profileApi.update(input));
}

export function useConversations() {
  return useQuery({ queryKey: queryKeys.conversations, queryFn: chatApi.conversations, staleTime: 15_000 });
}

export function useUnreadMessagesCount(): number {
  const { data } = useConversations();
  return (data ?? []).reduce((sum, c: Conversation) => sum + c.unreadCount, 0);
}

export function useNotifications() {
  return useQuery({ queryKey: queryKeys.notifications, queryFn: () => notificationsApi.list(0), staleTime: 15_000 });
}

export type MessagesData = InfiniteData<MessagesPage, string | undefined>;

export function useMessages(userId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(userId),
    queryFn: ({ pageParam }) => chatApi.messages(userId, pageParam),
    initialPageParam: undefined as string | undefined,
    // Pages go from newest to oldest; the cursor is the oldest message id.
    getNextPageParam: (last) => (last.hasMore && last.messages.length > 0 ? last.messages[0].id : undefined),
    staleTime: 0
  });
}

/** Inserts a message into the newest page if it is not there yet. */
export function appendMessage(data: MessagesData | undefined, message: Message): MessagesData | undefined {
  if (!data || data.pages.length === 0) {
    return data;
  }
  const exists = data.pages.some((p) => p.messages.some((m) => m.id === message.id));
  if (exists) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
}
