import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { profileApi, type UploadablePhoto } from "../api/profile";
import { chatApi, matchesApi, notificationsApi, safetyApi } from "../api/social";
import type { MeProfile, Message, MessagesPage, Preferences, ProfileInput, ReportReason } from "../api/types";

export const keys = {
  me: ["me", "profile"] as const,
  interests: ["interests"] as const,
  deck: ["discover"] as const,
  matches: ["matches"] as const,
  match: (id: string) => ["matches", id] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  profile: (userId: string) => ["profiles", userId] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: keys.me, queryFn: profileApi.me });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: async () => (await profileApi.interests()).interests, staleTime: Infinity });
}

export function useProfileCard(userId: string) {
  return useQuery({ queryKey: keys.profile(userId), queryFn: () => profileApi.view(userId) });
}

function useSetMe() {
  const qc = useQueryClient();
  return (me: MeProfile) => qc.setQueryData(keys.me, me);
}

export function useSaveProfile() {
  const setMe = useSetMe();
  return useMutation({ mutationFn: (input: ProfileInput) => profileApi.save(input), onSuccess: setMe });
}

export function useSavePreferences() {
  const setMe = useSetMe();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (prefs: Preferences) => profileApi.savePreferences(prefs),
    onSuccess: (me) => {
      setMe(me);
      void qc.invalidateQueries({ queryKey: keys.deck });
    }
  });
}

export function usePhotoMutations() {
  const qc = useQueryClient();
  const refresh = () => qc.invalidateQueries({ queryKey: keys.me });
  return {
    upload: useMutation({ mutationFn: (p: UploadablePhoto) => profileApi.uploadPhoto(p), onSuccess: refresh }),
    replace: useMutation({ mutationFn: (v: { id: string; photo: UploadablePhoto }) => profileApi.replacePhoto(v.id, v.photo), onSuccess: refresh }),
    remove: useMutation({ mutationFn: (id: string) => profileApi.deletePhoto(id), onSuccess: refresh }),
    reorder: useMutation({ mutationFn: (ids: string[]) => profileApi.reorderPhotos(ids), onSuccess: refresh })
  };
}

export function useMatches() {
  return useInfiniteQuery({
    queryKey: keys.matches,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    getNextPageParam: (last, pages) => (last.hasMore ? pages.reduce((n, p) => n + p.matches.length, 0) : undefined)
  });
}

export function useMatch(matchId: string) {
  return useQuery({ queryKey: keys.match(matchId), queryFn: () => matchesApi.get(matchId) });
}

export function useMessages(matchId: string) {
  return useInfiniteQuery({
    queryKey: keys.messages(matchId),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) => chatApi.messages(matchId, pageParam),
    getNextPageParam: (last) => last.nextBefore ?? undefined,
    staleTime: 10_000
  });
}

type MessagesCache = InfiniteData<MessagesPage, string | null>;

/** Inserts or replaces a message (matched by id or clientMessageId) at the newest end. */
export function upsertMessage(cache: MessagesCache | undefined, message: Message): MessagesCache | undefined {
  if (!cache) {
    return cache;
  }
  let replaced = false;
  const pages = cache.pages.map((page) => ({
    ...page,
    messages: page.messages.map((m) => {
      if (m.id === message.id || (message.clientMessageId && m.clientMessageId === message.clientMessageId)) {
        replaced = true;
        return message;
      }
      return m;
    })
  }));
  if (!replaced && pages.length > 0) {
    pages[0] = { ...pages[0], messages: [message, ...pages[0].messages] };
  }
  return { ...cache, pages };
}

export function markOwnMessagesRead(cache: MessagesCache | undefined, readerId: string, readAt: string): MessagesCache | undefined {
  if (!cache) {
    return cache;
  }
  return {
    ...cache,
    pages: cache.pages.map((page) => ({
      ...page,
      messages: page.messages.map((m) => (m.senderId !== readerId && !m.readAt && !m.pending ? { ...m, readAt } : m))
    }))
  };
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    getNextPageParam: (last, pages) => (last.hasMore ? pages.reduce((n, p) => n + p.notifications.length, 0) : undefined)
  });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: async () => (await safetyApi.blocks()).blocks });
}

/** Invalidates every list that may contain a user after block/report/unmatch. */
export function useSafetyMutations() {
  const qc = useQueryClient();
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: keys.matches });
    void qc.invalidateQueries({ queryKey: keys.blocks });
    void qc.invalidateQueries({ queryKey: keys.notifications });
  };
  return {
    block: useMutation({ mutationFn: (userId: string) => safetyApi.block(userId), onSuccess: refresh }),
    unblock: useMutation({ mutationFn: (userId: string) => safetyApi.unblock(userId), onSuccess: refresh }),
    report: useMutation({
      mutationFn: (v: { userId: string; reason: ReportReason; details: string }) => safetyApi.report(v.userId, v.reason, v.details),
      onSuccess: refresh
    }),
    unmatch: useMutation({ mutationFn: (matchId: string) => matchesApi.unmatch(matchId), onSuccess: refresh })
  };
}
