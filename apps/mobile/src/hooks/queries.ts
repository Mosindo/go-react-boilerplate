import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, discoveryApi, notificationsApi, photosApi, profileApi, safetyApi } from "../api/endpoints";
import type { ChatSummary, Match, Message, MessagesPage, MyProfile, Photo, Preferences, ProfileInput, ReportReason } from "../api/types";
import { queryKeys } from "./queryClient";

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.me });
}

export function useInterestsCatalog() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

/** Every profile mutation returns the fresh profile: write it to the cache. */
function useProfileMutation<TInput>(fn: (input: TInput) => Promise<MyProfile>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (profile) => {
      qc.setQueryData(queryKeys.myProfile, profile);
      void qc.invalidateQueries({ queryKey: queryKeys.discovery });
    }
  });
}

export const useSaveProfile = () => useProfileMutation((input: ProfileInput) => profileApi.save(input));
export const useSavePreferences = () => useProfileMutation((prefs: Preferences) => profileApi.savePreferences(prefs));
export const useSaveInterests = () => useProfileMutation((slugs: string[]) => profileApi.saveInterests(slugs));
export const useSavePrivacy = () =>
  useProfileMutation((privacy: { isDiscoverable?: boolean; showDistance?: boolean }) => profileApi.savePrivacy(privacy));
export const useSaveLocation = () =>
  useProfileMutation((loc: { latitude: number; longitude: number; city?: string }) =>
    profileApi.saveLocation(loc.latitude, loc.longitude, loc.city)
  );
export const useClearLocation = () => useProfileMutation(() => profileApi.clearLocation());

/** Photo mutations return the photo list: patch it into the cached profile. */
export function usePhotoMutation<TInput>(fn: (input: TInput) => Promise<Photo[]>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (photos) => {
      qc.setQueryData<MyProfile>(queryKeys.myProfile, (prev) => {
        if (!prev) {
          return prev;
        }
        const missing: MyProfile["missing"] = prev.missing.filter((m) => m !== "photos");
        if (photos.length === 0) {
          missing.push("photos");
        }
        return { ...prev, photos, missing, isComplete: missing.length === 0 };
      });
    }
  });
}

export const useUploadPhoto = () => usePhotoMutation((uri: string) => photosApi.upload(uri));
export const useReplacePhoto = () =>
  usePhotoMutation((input: { photoId: string; uri: string }) => photosApi.replace(input.photoId, input.uri));
export const useDeletePhoto = () => usePhotoMutation((photoId: string) => photosApi.remove(photoId));
export const useReorderPhotos = () => usePhotoMutation((photoIds: string[]) => photosApi.reorder(photoIds));

export function useProfileCard(userId: string) {
  return useQuery({ queryKey: queryKeys.profile(userId), queryFn: () => discoveryApi.profile(userId) });
}

/** Polling interval used only as a fallback while the realtime socket is down. */
export type Polling = number | false;

export function useMatches(refetchInterval: Polling = false) {
  return useQuery({ queryKey: queryKeys.matches, queryFn: discoveryApi.matches, refetchInterval });
}

export function useChats(refetchInterval: Polling = false) {
  return useQuery({ queryKey: queryKeys.chats, queryFn: chatApi.list, refetchInterval });
}

export function useMessages(userId: string, refetchInterval: Polling = false) {
  return useInfiniteQuery({
    refetchInterval,
    queryKey: queryKeys.messages(userId),
    queryFn: ({ pageParam }) => chatApi.messages(userId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage: MessagesPage) => (lastPage.hasMore ? lastPage.messages[0]?.id : undefined)
  });
}

export function useSendMessage(userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (content: string) => chatApi.send(userId, content),
    onSuccess: (message) => {
      appendMessage(qc, userId, message);
      upsertChatSummary(qc, userId, message, false);
    }
  });
}

type QC = ReturnType<typeof useQueryClient>;
type MessagesData = { pages: MessagesPage[]; pageParams: (string | undefined)[] };

/** Adds a message to the newest page of a thread (deduplicated by id). */
export function appendMessage(qc: QC, otherUserId: string, message: Message) {
  qc.setQueryData<MessagesData>(queryKeys.messages(otherUserId), (prev) => {
    if (!prev || prev.pages.length === 0) {
      return prev;
    }
    if (prev.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
      return prev;
    }
    const [newest, ...older] = prev.pages;
    return { ...prev, pages: [{ ...newest, messages: [...newest.messages, message] }, ...older] };
  });
}

export function setOtherReadAt(qc: QC, otherUserId: string, readAt: string) {
  qc.setQueryData<MessagesData>(queryKeys.messages(otherUserId), (prev) => {
    if (!prev || prev.pages.length === 0) {
      return prev;
    }
    const [newest, ...older] = prev.pages;
    return { ...prev, pages: [{ ...newest, otherLastReadAt: readAt }, ...older] };
  });
}

/** Moves a conversation to the top of the list with its latest message. */
export function upsertChatSummary(qc: QC, otherUserId: string, message: Message, incoming: boolean) {
  const known = qc.getQueryData<ChatSummary[]>(queryKeys.chats);
  const existing = known?.find((c) => c.user.userId === otherUserId);
  if (!known || !existing) {
    void qc.invalidateQueries({ queryKey: queryKeys.chats });
    void qc.invalidateQueries({ queryKey: queryKeys.matches });
    return;
  }
  const updated: ChatSummary = {
    ...existing,
    lastMessage: message,
    unreadCount: incoming ? existing.unreadCount + 1 : existing.unreadCount
  };
  qc.setQueryData<ChatSummary[]>(queryKeys.chats, [updated, ...known.filter((c) => c.user.userId !== otherUserId)]);
  qc.setQueryData<Match[]>(queryKeys.matches, (prev) =>
    prev?.map((m) => (m.user.userId === otherUserId ? { ...m, hasMessages: true } : m))
  );
}

export function useMarkRead(userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => chatApi.markRead(userId),
    onSuccess: () => {
      qc.setQueryData<ChatSummary[]>(queryKeys.chats, (prev) =>
        prev?.map((c) => (c.user.userId === userId ? { ...c, unreadCount: 0 } : c))
      );
    }
  });
}

/** Removes every trace of a relationship from the caches (unmatch, block). */
export function forgetRelationship(qc: QC, userId: string) {
  qc.setQueryData<Match[]>(queryKeys.matches, (prev) => prev?.filter((m) => m.user.userId !== userId));
  qc.setQueryData<ChatSummary[]>(queryKeys.chats, (prev) => prev?.filter((c) => c.user.userId !== userId));
  qc.removeQueries({ queryKey: queryKeys.messages(userId) });
  qc.removeQueries({ queryKey: queryKeys.profile(userId) });
}

export function useUnmatch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => discoveryApi.unmatch(userId),
    onSuccess: (_, userId) => forgetRelationship(qc, userId)
  });
}

export function useBlock() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => safetyApi.block(userId),
    onSuccess: (_, userId) => {
      forgetRelationship(qc, userId);
      void qc.invalidateQueries({ queryKey: queryKeys.blocks });
    }
  });
}

export function useReport() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { userId: string; reason: ReportReason; details: string }) =>
      safetyApi.report(input.userId, input.reason, input.details),
    onSuccess: (_, input) => {
      forgetRelationship(qc, input.userId);
      void qc.invalidateQueries({ queryKey: queryKeys.blocks });
    }
  });
}

export function useBlocks() {
  return useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
}

export function useUnblock() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => safetyApi.unblock(userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.blocks })
  });
}

export function useNotifications(refetchInterval: Polling = false) {
  return useQuery({ queryKey: queryKeys.notifications, queryFn: notificationsApi.list, refetchInterval });
}
