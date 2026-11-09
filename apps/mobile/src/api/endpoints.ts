import { request } from "./client";
import type {
  AppNotification,
  AuthResponse,
  BlockedUser,
  Conversation,
  Interest,
  Message,
  OwnProfile,
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
    request<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => request<unknown>("/auth/forgot-password", { method: "POST", body: { email }, auth: false }),
  resetPassword: (token: string, password: string) =>
    request<void>("/auth/reset-password", { method: "POST", body: { token, password }, auth: false }),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  me: () => request<OwnProfile>("/me/profile"),
  save: (input: ProfileInput) => request<OwnProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => request<Preferences>("/me/preferences", { method: "PUT", body: prefs }),
  setLocation: (latitude: number, longitude: number) => request<void>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  get: (userId: string) => request<PublicProfile>(`/profiles/${userId}`)
};

export type UploadFile = { uri: string; name: string; type: string };

function photoForm(file: UploadFile): FormData {
  const form = new FormData();
  // React Native's FormData accepts {uri,name,type} descriptors in place of Blob.
  form.append("photo", file as unknown as Blob);
  return form;
}

export const photoApi = {
  list: () => request<{ photos: Photo[] }>("/me/photos").then((r) => r.photos),
  upload: (file: UploadFile) => request<Photo>("/me/photos", { method: "POST", form: photoForm(file) }),
  replace: (id: string, file: UploadFile) =>
    request<{ photos: Photo[] }>(`/me/photos/${id}`, { method: "PUT", form: photoForm(file) }).then((r) => r.photos),
  remove: (id: string) => request<void>(`/me/photos/${id}`, { method: "DELETE" }),
  makePrimary: (id: string) => request<{ photos: Photo[] }>(`/me/photos/${id}/primary`, { method: "PUT" }).then((r) => r.photos),
  reorder: (ids: string[]) => request<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { ids } }).then((r) => r.photos)
};

export const matchingApi = {
  discover: (limit = 10) => request<{ profiles: PublicProfile[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetUserId: string, action: SwipeAction) => request<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  resetPasses: () => request<{ reset: number }>("/discover/reset", { method: "POST" }),
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: (before?: string) =>
    request<{ conversations: Conversation[] }>(`/conversations?limit=30${before ? `&before=${encodeURIComponent(before)}` : ""}`).then(
      (r) => r.conversations
    ),
  messages: (conversationId: string, before?: string) =>
    request<{ messages: Message[] }>(`/conversations/${conversationId}/messages?limit=30${before ? `&before=${before}` : ""}`).then(
      (r) => r.messages
    ),
  send: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => request<void>(`/conversations/${conversationId}/read`, { method: "POST" }),
  hide: (conversationId: string) => request<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const safetyApi = {
  blocks: () => request<{ blocks: BlockedUser[] }>("/me/blocks").then((r) => r.blocks),
  block: (userId: string) => request<void>(`/users/${userId}/block`, { method: "PUT" }),
  unblock: (userId: string) => request<void>(`/users/${userId}/block`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    request<void>(`/users/${userId}/report`, { method: "POST", body: { reason, details, block } })
};

export const notificationApi = {
  list: (before?: string) =>
    request<{ notifications: AppNotification[]; unreadCount: number }>(
      `/notifications?limit=30${before ? `&before=${encodeURIComponent(before)}` : ""}`
    ),
  markRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request<void>("/notifications/read", { method: "POST" })
};
