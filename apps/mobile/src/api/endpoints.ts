import { request, uploadPhoto } from "./client";
import type {
  AuthResponse,
  BlockedUser,
  ConversationsPage,
  Interest,
  Message,
  MessagesPage,
  NotificationsPage,
  OwnProfile,
  Photo,
  Preferences,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResponse,
  UpsertProfileInput,
  User,
} from "./types";

const q = (params: Record<string, string | number | undefined>): string => {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== "")
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`);
  return parts.length ? `?${parts.join("&")}` : "";
};

export const authApi = {
  register: (email: string, password: string) => request<AuthResponse>("/auth/register", { method: "POST", body: { email, password } }),
  login: (email: string, password: string) => request<AuthResponse>("/auth/login", { method: "POST", body: { email, password } }),
  logout: (refreshToken: string) => request<void>("/auth/logout", { method: "POST", body: { refreshToken } }),
  forgot: (email: string) => request<void>("/auth/password/forgot", { method: "POST", body: { email } }),
  reset: (email: string, code: string, newPassword: string) =>
    request<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword } }),
  me: () => request<User>("/me"),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } }),
};

export const profileApi = {
  me: () => request<OwnProfile>("/profile/me"),
  save: (input: UpsertProfileInput) => request<OwnProfile>("/profile/me", { method: "PUT", body: input }),
  savePreferences: (p: Preferences) => request<OwnProfile>("/profile/preferences", { method: "PUT", body: p }),
  saveLocation: (latitude: number, longitude: number, city?: string) =>
    request<OwnProfile>("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  interests: () => request<{ interests: Interest[] }>("/interests"),
  view: (id: string) => request<PublicProfile>(`/profiles/${id}`),
};

export const photosApi = {
  upload: (uri: string, mimeType?: string) => uploadPhoto<Photo>("/photos", "POST", { uri, mimeType }),
  replace: (id: string, uri: string, mimeType?: string) => uploadPhoto<Photo>(`/photos/${id}`, "PUT", { uri, mimeType }),
  remove: (id: string) => request<void>(`/photos/${id}`, { method: "DELETE" }),
  makePrimary: (id: string) => request<{ photos: Photo[] }>(`/photos/${id}/primary`, { method: "PUT" }),
  reorder: (ids: string[]) => request<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { ids } }),
};

export const discoveryApi = {
  deck: (limit = 10) => request<{ profiles: PublicProfile[] }>(`/discover${q({ limit })}`),
  likesReceived: () => request<{ profiles: PublicProfile[] }>("/likes/received?limit=50"),
  swipe: (userId: string, action: SwipeAction) => request<SwipeResponse>("/swipes", { method: "POST", body: { userId, action } }),
};

export const chatApi = {
  conversations: (cursor?: string) => request<ConversationsPage>(`/conversations${q({ cursor, limit: 30 })}`),
  messages: (id: string, before?: string) => request<MessagesPage>(`/conversations/${id}/messages${q({ before, limit: 40 })}`),
  send: (id: string, body: string) => request<Message>(`/conversations/${id}/messages`, { method: "POST", body: { body } }),
  markRead: (id: string) => request<void>(`/conversations/${id}/read`, { method: "POST" }),
  hide: (id: string) => request<void>(`/conversations/${id}`, { method: "DELETE" }),
  unmatch: (id: string) => request<void>(`/conversations/${id}/unmatch`, { method: "POST" }),
};

export const notificationsApi = {
  list: (cursor?: string) => request<NotificationsPage>(`/notifications${q({ cursor, limit: 30 })}`),
  markRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request<void>("/notifications/read-all", { method: "POST" }),
};

export const safetyApi = {
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => request<{ blocks: BlockedUser[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details, alsoBlock } }),
};
