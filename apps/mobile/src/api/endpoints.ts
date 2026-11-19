import { request } from "./client";
import type {
  AuthResponse,
  AuthUser,
  BlockedUser,
  Conversation,
  ConversationsPage,
  Interest,
  Message,
  MessagesPage,
  NotificationsPage,
  OwnProfile,
  Photo,
  Preferences,
  ProfileCard,
  ProfileInput,
  ReportReason,
  SwipeResponse
} from "./types";

const q = (params: Record<string, string | number | undefined | null>): string => {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== "")
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  return parts.length ? `?${parts.join("&")}` : "";
};

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => request<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    request<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  me: () => request<AuthUser>("/me"),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  get: () => request<OwnProfile>("/profile"),
  save: (input: ProfileInput) => request<OwnProfile>("/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => request<Preferences>("/profile/preferences", { method: "PUT", body: prefs }),
  saveLocation: (latitude: number, longitude: number, city?: string) =>
    request<void>("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => request<void>("/profile/location", { method: "DELETE" }),
  view: (userId: string) => request<ProfileCard>(`/profiles/${userId}`)
};

export const photosApi = {
  upload: (form: FormData) => request<Photo>("/photos", { method: "POST", form, timeoutMs: 60_000 }),
  replace: (photoId: string, form: FormData) =>
    request<{ photos: Photo[] }>(`/photos/${photoId}`, { method: "PUT", form, timeoutMs: 60_000 }).then((r) => r.photos),
  remove: (photoId: string) => request<{ photos: Photo[] }>(`/photos/${photoId}`, { method: "DELETE" }).then((r) => r.photos),
  reorder: (photoIds: string[]) =>
    request<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } }).then((r) => r.photos)
};

export const discoveryApi = {
  next: (limit = 10) => request<{ profiles: ProfileCard[] }>(`/discovery${q({ limit })}`).then((r) => r.profiles),
  like: (userId: string) => request<SwipeResponse>(`/discovery/${userId}/like`, { method: "POST" }),
  pass: (userId: string) => request<SwipeResponse>(`/discovery/${userId}/pass`, { method: "POST" }),
  likesReceived: () => request<{ profiles: ProfileCard[] }>("/likes/received").then((r) => r.profiles),
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  list: (cursor?: string | null) => request<ConversationsPage>(`/conversations${q({ cursor, limit: 20 })}`),
  get: (id: string) => request<Conversation>(`/conversations/${id}`),
  messages: (id: string, cursor?: string | null) => request<MessagesPage>(`/conversations/${id}/messages${q({ cursor, limit: 30 })}`),
  send: (id: string, body: string) => request<Message>(`/conversations/${id}/messages`, { method: "POST", body: { body } }),
  markRead: (id: string) => request<void>(`/conversations/${id}/read`, { method: "POST" }),
  clear: (id: string) => request<void>(`/conversations/${id}`, { method: "DELETE" })
};

export const notificationsApi = {
  list: (cursor?: string | null) => request<NotificationsPage>(`/notifications${q({ cursor, limit: 20 })}`),
  unreadCount: () => request<{ unreadCount: number }>("/notifications/unread-count").then((r) => r.unreadCount),
  markAllRead: () => request<void>("/notifications/read", { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => request<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details } })
};

export const realtimeApi = {
  ticket: () => request<{ ticket: string }>("/realtime/ticket", { method: "POST" }).then((r) => r.ticket)
};
