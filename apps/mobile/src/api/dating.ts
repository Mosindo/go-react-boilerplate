import { Platform } from "react-native";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  BlockedUser,
  Gender,
  Interest,
  MatchSummary,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Photo,
  Preferences,
  PublicProfile,
  ReadReceipt,
  ReportReason,
  Settings,
  SwipeAction,
  SwipeResult
} from "./types";

export type ProfilePatch = Partial<{
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
}>;

export const profileApi = {
  mine: () => apiRequest<MyProfile>(endpoints.profile.mine),
  update: (patch: ProfilePatch) => apiRequest<MyProfile>(endpoints.profile.mine, { method: "PATCH", body: patch }),
  updatePreferences: (prefs: Preferences) => apiRequest<MyProfile>(endpoints.profile.preferences, { method: "PUT", body: prefs }),
  updateSettings: (settings: Settings) => apiRequest<MyProfile>(endpoints.profile.settings, { method: "PUT", body: settings }),
  updateLocation: (latitude: number, longitude: number, city?: string) =>
    apiRequest<MyProfile>(endpoints.profile.location, { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => apiRequest<MyProfile>(endpoints.profile.location, { method: "DELETE" }),
  interests: () => apiRequest<{ interests: Interest[] }>(endpoints.profile.interests).then((r) => r.interests),
  publicProfile: (userId: string) => apiRequest<PublicProfile>(endpoints.profile.public(userId))
};

async function photoFormData(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string) =>
    apiRequest<{ photos: Photo[] }>(endpoints.profile.photos, { method: "POST", formData: await photoFormData(uri), timeoutMs: 60_000 }),
  replace: async (photoId: string, uri: string) =>
    apiRequest<{ photos: Photo[] }>(endpoints.profile.photo(photoId), {
      method: "PUT",
      formData: await photoFormData(uri),
      timeoutMs: 60_000
    }),
  remove: (photoId: string) => apiRequest<{ photos: Photo[] }>(endpoints.profile.photo(photoId), { method: "DELETE" }),
  reorder: (photoIds: string[]) => apiRequest<{ photos: Photo[] }>(endpoints.profile.photosOrder, { method: "PUT", body: { photoIds } })
};

export const discoveryApi = {
  deck: (limit = 10) => apiRequest<{ profiles: PublicProfile[] }>(endpoints.discovery.deck(limit)).then((r) => r.profiles),
  swipe: (targetUserId: string, action: SwipeAction) =>
    apiRequest<SwipeResult>(endpoints.discovery.swipe, { method: "POST", body: { targetUserId, action } })
};

export const matchesApi = {
  list: (offset = 0, limit = 30) => apiRequest<{ matches: MatchSummary[]; hasMore: boolean }>(endpoints.matches.list(limit, offset)),
  unmatch: (matchId: string) => apiRequest<void>(endpoints.matches.remove(matchId), { method: "DELETE" })
};

export const chatApi = {
  messages: (conversationId: string, before?: string) => apiRequest<MessagesPage>(endpoints.chat.messages(conversationId, before)),
  send: (conversationId: string, content: string) =>
    apiRequest<Message>(endpoints.chat.send(conversationId), { method: "POST", body: { content } }),
  markRead: (conversationId: string) => apiRequest<ReadReceipt>(endpoints.chat.read(conversationId), { method: "POST" })
};

export const safetyApi = {
  blocks: () => apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks).then((r) => r.blocks),
  block: (userId: string) => apiRequest<void>(endpoints.safety.blocks, { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest<void>(endpoints.safety.unblock(userId), { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    apiRequest<{ id: string; blocked: boolean }>(endpoints.safety.reports, { method: "POST", body: { userId, reason, details } })
};

export const notificationsApi = {
  list: (offset = 0) => apiRequest<NotificationsPage>(endpoints.notifications.list(offset)),
  unreadCount: () => apiRequest<{ unreadCount: number }>(endpoints.notifications.unreadCount).then((r) => r.unreadCount),
  readAll: () => apiRequest<void>(endpoints.notifications.readAll, { method: "POST" }),
  read: (id: string) => apiRequest<void>(endpoints.notifications.read(id), { method: "POST" })
};
