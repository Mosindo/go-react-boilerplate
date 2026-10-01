import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  AppNotification,
  BlockedUser,
  ConversationSummary,
  InterestRef,
  MatchSummary,
  Message,
  OwnProfile,
  Photo,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResponse
} from "./types";

/** React Native's FormData accepts a { uri, name, type } descriptor in place of a Blob. */
export type UploadFile = { uri: string; name: string; type: string };

function photoForm(file: UploadFile): FormData {
  const form = new FormData();
  form.append("photo", file as unknown as Blob);
  return form;
}

export const profileApi = {
  getOwn: () => apiRequest<OwnProfile>(endpoints.profile.own),
  save: (input: ProfileInput) => apiRequest<OwnProfile>(endpoints.profile.own, { method: "PUT", body: input }),
  getPreferences: () => apiRequest<Preferences>(endpoints.profile.preferences),
  savePreferences: (input: Preferences) => apiRequest<Preferences>(endpoints.profile.preferences, { method: "PUT", body: input }),
  setLocation: (latitude: number, longitude: number, city?: string) =>
    apiRequest(endpoints.profile.location, { method: "PUT", body: { latitude, longitude, city: city ?? "" } }),
  clearLocation: () => apiRequest(endpoints.profile.location, { method: "DELETE" }),
  interests: () => apiRequest<{ interests: InterestRef[] }>(endpoints.profile.interests).then((r) => r.interests),
  getPublic: (userId: string) => apiRequest<PublicProfile>(endpoints.profile.public(userId))
};

export const photoApi = {
  list: () => apiRequest<{ photos: Photo[] }>(endpoints.photos.list).then((r) => r.photos),
  add: (file: UploadFile) => apiRequest<Photo>(endpoints.photos.list, { method: "POST", form: photoForm(file), timeoutMs: 60_000 }),
  replace: (photoId: string, file: UploadFile) =>
    apiRequest<Photo>(endpoints.photos.item(photoId), { method: "PUT", form: photoForm(file), timeoutMs: 60_000 }),
  remove: (photoId: string) => apiRequest(endpoints.photos.item(photoId), { method: "DELETE" }),
  reorder: (photoIds: string[]) =>
    apiRequest<{ photos: Photo[] }>(endpoints.photos.order, { method: "PUT", body: { photoIds } }).then((r) => r.photos)
};

export const discoverApi = {
  next: (limit = 10) => apiRequest<{ profiles: PublicProfile[] }>(endpoints.discover.list, { query: { limit } }).then((r) => r.profiles),
  swipe: (userId: string, action: SwipeAction) =>
    apiRequest<SwipeResponse>(endpoints.discover.swipe, { method: "POST", body: { userId, action } }),
  matches: (limit = 30, offset = 0) =>
    apiRequest<{ matches: MatchSummary[] }>(endpoints.discover.matches, { query: { limit, offset } }).then((r) => r.matches),
  unmatch: (matchId: string) => apiRequest(endpoints.discover.unmatch(matchId), { method: "DELETE" })
};

export const chatApi = {
  conversations: (limit = 30, offset = 0) =>
    apiRequest<{ conversations: ConversationSummary[]; totalUnread: number }>(endpoints.chat.conversations, { query: { limit, offset } }),
  messages: (conversationId: string, before?: string, limit = 30) =>
    apiRequest<{ messages: Message[]; hasMore: boolean }>(endpoints.chat.messages(conversationId), { query: { before, limit } }),
  send: (conversationId: string, body: string) =>
    apiRequest<Message>(endpoints.chat.messages(conversationId), { method: "POST", body: { body } }),
  markRead: (conversationId: string) => apiRequest<{ marked: number }>(endpoints.chat.read(conversationId), { method: "POST" }),
  clear: (conversationId: string) => apiRequest(endpoints.chat.clear(conversationId), { method: "DELETE" })
};

export const notificationApi = {
  list: (limit = 30, offset = 0) =>
    apiRequest<{ notifications: AppNotification[]; unreadCount: number }>(endpoints.notifications.list, { query: { limit, offset } }),
  markRead: (notificationId: string) => apiRequest<AppNotification>(endpoints.notifications.markRead(notificationId), { method: "POST" }),
  markAllRead: () => apiRequest<{ marked: number }>(endpoints.notifications.readAll, { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => apiRequest(endpoints.safety.blocks, { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest(endpoints.safety.unblock(userId), { method: "DELETE" }),
  blocks: () => apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks).then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    apiRequest<{ id: string }>(endpoints.safety.reports, { method: "POST", body: { userId, reason, details, block } })
};

export const realtimeApi = {
  ticket: () => apiRequest<{ ticket: string; expiresIn: number }>(endpoints.realtime.ticket, { method: "POST" })
};
