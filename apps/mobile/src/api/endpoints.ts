import { api } from "./client";
import type {
  AppNotification,
  AuthResponse,
  BlockedUser,
  Card,
  Conversation,
  Interest,
  Me,
  Message,
  OwnProfile,
  Photo,
  Preferences,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult,
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api.public<AuthResponse>("POST", "/auth/register", { email, password }),
  login: (email: string, password: string) =>
    api.public<AuthResponse>("POST", "/auth/login", { email, password }),
  logout: (refreshToken: string) => api.public<void>("POST", "/auth/logout", { refreshToken }),
  forgotPassword: (email: string) => api.public<void>("POST", "/auth/forgot-password", { email }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api.public<void>("POST", "/auth/reset-password", { email, code, newPassword }),
  me: () => api.get<Me>("/me"),
  deleteAccount: (password: string) => api.del<void>("/me", { password }),
};

export const profileApi = {
  interests: () => api.get<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  own: () => api.get<OwnProfile>("/me/profile"),
  save: (input: ProfileInput) => api.put<OwnProfile>("/me/profile", input),
  savePreferences: (prefs: Preferences) => api.put<Preferences>("/me/preferences", prefs),
  setLocation: (latitude: number, longitude: number) =>
    api.put<void>("/me/location", { latitude, longitude }),
  clearLocation: () => api.del<void>("/me/location"),
  card: (userId: string) => api.get<Card>(`/profiles/${userId}`),
};

export const photoApi = {
  upload: (form: FormData) => api.upload<Photo>("POST", "/me/photos", form),
  replace: (id: string, form: FormData) => api.upload<Photo>("PUT", `/me/photos/${id}`, form),
  remove: (id: string) => api.del<void>(`/me/photos/${id}`),
  reorder: (photoIds: string[]) => api.put<void>("/me/photos/order", { photoIds }),
};

export const discoverApi = {
  feed: (limit = 10) =>
    api.get<{ profiles: Card[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetId: string, action: SwipeAction) =>
    api.post<SwipeResult>("/swipes", { targetId, action }),
  unmatch: (matchId: string) => api.del<void>(`/matches/${matchId}`),
};

export const chatApi = {
  conversations: (limit = 50) =>
    api
      .get<{ conversations: Conversation[] }>(`/conversations?limit=${limit}`)
      .then((r) => r.conversations),
  unread: () => api.get<{ unread: number }>("/conversations/unread-count").then((r) => r.unread),
  messages: (conversationId: string, before?: string, limit = 40) =>
    api
      .get<{ messages: Message[] }>(
        `/conversations/${conversationId}/messages?limit=${limit}${before ? `&before=${before}` : ""}`,
      )
      .then((r) => r.messages),
  send: (conversationId: string, content: string) =>
    api.post<Message>(`/conversations/${conversationId}/messages`, { content }),
  markRead: (conversationId: string) =>
    api.post<{ marked: number }>(`/conversations/${conversationId}/read`),
  hide: (conversationId: string) => api.del<void>(`/conversations/${conversationId}`),
};

export const safetyApi = {
  block: (userId: string) => api.post<void>("/blocks", { userId }),
  unblock: (userId: string) => api.del<void>(`/blocks/${userId}`),
  blocked: () => api.get<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    api.post<void>("/reports", { userId, reason, details, block }),
};

export const notificationApi = {
  list: (limit = 30) =>
    api
      .get<{ notifications: AppNotification[] }>(`/notifications?limit=${limit}`)
      .then((r) => r.notifications),
  unread: () => api.get<{ unread: number }>("/notifications/unread-count").then((r) => r.unread),
  markRead: (id: string) => api.post<AppNotification>(`/notifications/${id}/read`),
  markAllRead: () => api.post<void>("/notifications/read-all"),
};
