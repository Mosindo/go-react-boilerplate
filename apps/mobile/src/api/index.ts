import { Platform } from "react-native";
import { api } from "./client";
import type {
  AuthResponse,
  BlockedUser,
  Conversation,
  Gender,
  Interest,
  MatchView,
  Message,
  AppNotification,
  OwnProfile,
  Photo,
  Preferences,
  PublicProfile,
  ReportReason,
  SwipeResult
} from "./types";

export * from "./types";
export { ApiError, API_BASE_URL, isApiError } from "./client";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthResponse>("/auth/register", { method: "POST", auth: false, body: { email, password } }),
  login: (email: string, password: string) =>
    api<AuthResponse>("/auth/login", { method: "POST", auth: false, body: { email, password } }),
  logout: (refreshToken: string) =>
    api("/auth/logout", { method: "POST", auth: false, body: { refreshToken } }),
  forgotPassword: (email: string) =>
    api("/auth/password/forgot", { method: "POST", auth: false, body: { email } }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api("/auth/password/reset", { method: "POST", auth: false, body: { email, code, newPassword } }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api("/me", { method: "DELETE", body: { password } })
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  interests: string[];
};

export const profileApi = {
  get: () => api<OwnProfile>("/me/profile"),
  save: (input: ProfileInput) => api<OwnProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => api<OwnProfile>("/me/preferences", { method: "PUT", body: prefs }),
  saveLocation: (latitude: number, longitude: number, city: string) =>
    api<OwnProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  savePrivacy: (isVisible: boolean, showDistance: boolean) =>
    api<OwnProfile>("/me/privacy", { method: "PUT", body: { isVisible, showDistance } }),
  interests: () => api<{ interests: Interest[] }>("/interests").then((r) => r.interests)
};

export type PickedImage = {
  uri: string;
  fileName?: string | null;
  mimeType?: string | null;
  file?: Blob | null;
};

async function toFormData(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  const name = image.fileName || "photo.jpg";
  if (Platform.OS === "web") {
    const blob = image.file ?? (await (await fetch(image.uri)).blob());
    form.append("file", blob, name);
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor.
    form.append("file", { uri: image.uri, name, type: image.mimeType || "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  list: () => api<{ photos: Photo[] }>("/me/photos").then((r) => r.photos),
  add: async (image: PickedImage) =>
    api<{ photos: Photo[] }>("/me/photos", { method: "POST", form: await toFormData(image) }).then(
      (r) => r.photos
    ),
  replace: async (id: string, image: PickedImage) =>
    api<{ photos: Photo[] }>(`/me/photos/${id}`, { method: "PUT", form: await toFormData(image) }).then(
      (r) => r.photos
    ),
  remove: (id: string) =>
    api<{ photos: Photo[] }>(`/me/photos/${id}`, { method: "DELETE" }).then((r) => r.photos),
  reorder: (ids: string[]) =>
    api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { ids } }).then((r) => r.photos)
};

export const discoverApi = {
  next: (limit = 10) =>
    api<{ profiles: PublicProfile[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  swipe: (userId: string, action: "like" | "pass") =>
    api<SwipeResult>("/discover/swipes", { method: "POST", body: { userId, action } }),
  match: (id: string) => api<MatchView>(`/matches/${id}`),
  unmatch: (id: string) => api(`/matches/${id}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: (cursor?: string) =>
    api<{ conversations: Conversation[]; nextCursor?: string }>(
      `/conversations?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`
    ),
  messages: (matchId: string, before?: number) =>
    api<{ messages: Message[]; hasMore: boolean }>(
      `/conversations/${matchId}/messages?limit=30${before ? `&before=${before}` : ""}`
    ),
  send: (matchId: string, body: string) =>
    api<Message>(`/conversations/${matchId}/messages`, { method: "POST", body: { body } }),
  markRead: (matchId: string) => api<{ count: number }>(`/conversations/${matchId}/read`, { method: "POST" }),
  clear: (matchId: string) => api(`/conversations/${matchId}`, { method: "DELETE" })
};

export const safetyApi = {
  block: (userId: string) => api("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string) =>
    api("/reports", { method: "POST", body: { userId, reason, details } })
};

export const notificationsApi = {
  list: (cursor?: string) =>
    api<{ notifications: AppNotification[]; unreadCount: number; nextCursor?: string }>(
      `/notifications?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`
    ),
  unreadCount: () => api<{ unreadCount: number }>("/notifications/unread-count").then((r) => r.unreadCount),
  markRead: (id: string) => api(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};
