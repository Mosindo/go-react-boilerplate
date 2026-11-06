import { api } from "./client";
import type {
  AuthResponse,
  AuthUser,
  BlockedUser,
  ConversationsPage,
  Interest,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Photo,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthResponse>("/auth/register", { method: "POST", auth: false, body: { email, password } }),
  login: (email: string, password: string) =>
    api<AuthResponse>("/auth/login", { method: "POST", auth: false, body: { email, password } }),
  logout: (refreshToken: string) =>
    api<void>("/auth/logout", { method: "POST", auth: false, body: { refreshToken } }),
  me: () => api<AuthUser>("/me"),
  forgot: (email: string) =>
    api<void>("/auth/password/forgot", { method: "POST", auth: false, body: { email } }),
  reset: (email: string, code: string, newPassword: string) =>
    api<void>("/auth/password/reset", {
      method: "POST",
      auth: false,
      body: { email, code, newPassword }
    }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  mine: () => api<MyProfile>("/me/profile"),
  save: (input: ProfileInput) => api<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) =>
    api<Preferences>("/me/preferences", { method: "PUT", body: prefs }),
  setPrivacy: (discoverable: boolean, showDistance: boolean) =>
    api<MyProfile>("/me/privacy", { method: "PUT", body: { discoverable, showDistance } }),
  interests: () => api<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  view: (userId: string) => api<PublicProfile>(`/profiles/${userId}`)
};

export const photoApi = {
  list: () => api<{ photos: Photo[] }>("/me/photos").then((r) => r.photos),
  upload: (form: FormData) => api<Photo>("/me/photos", { method: "POST", form }),
  replace: (id: string, form: FormData) => api<Photo>(`/me/photos/${id}`, { method: "PUT", form }),
  remove: (id: string) => api<void>(`/me/photos/${id}`, { method: "DELETE" }),
  reorder: (ids: string[]) =>
    api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { ids } }).then(
      (r) => r.photos
    )
};

export const discoverApi = {
  next: (limit = 10) =>
    api<{ profiles: PublicProfile[] }>("/discover", { query: { limit } }).then((r) => r.profiles),
  swipe: (targetId: string, action: SwipeAction) =>
    api<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } }),
  restorePasses: () => api<{ restored: number }>("/discover/passes", { method: "DELETE" }),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: (before?: string) =>
    api<ConversationsPage>("/conversations", { query: { before, limit: 30 } }),
  messages: (conversationId: string, before?: string) =>
    api<MessagesPage>(`/conversations/${conversationId}/messages`, {
      query: { before, limit: 30 }
    }),
  send: (conversationId: string, body: string) =>
    api<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) =>
    api<void>(`/conversations/${conversationId}/read`, { method: "POST" }),
  hide: (conversationId: string) =>
    api<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const safetyApi = {
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    api<void>("/reports", { method: "POST", body: { userId, reason, details, block } })
};

export const notificationsApi = {
  list: (before?: string) =>
    api<NotificationsPage>("/notifications", { query: { before, limit: 20 } }),
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: () => api<{ ticket: string }>("/ws/ticket", { method: "POST" }).then((r) => r.ticket)
};
