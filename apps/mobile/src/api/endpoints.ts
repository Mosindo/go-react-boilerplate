import { Platform } from "react-native";
import { api } from "./client";
import type {
  AuthSession,
  AuthUser,
  Card,
  Conversation,
  Interest,
  Message,
  NotificationItem,
  Page,
  PhotoRef,
  Preferences,
  ProfileBundle,
  ProfileInput,
  ReportReason,
  SwipeResult
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (token: string, password: string) =>
    api("/auth/password/reset", { method: "POST", body: { token, password }, auth: false }),
  deleteAccount: (password: string) => api("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  get: () => api<ProfileBundle>("/me/profile"),
  save: (input: ProfileInput) => api<ProfileBundle>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => api<ProfileBundle>("/me/preferences", { method: "PUT", body: prefs }),
  interests: () => api<{ items: Interest[] }>("/interests")
};

type UploadFile = { uri: string; mimeType?: string | null; fileName?: string | null };

async function buildForm(file: UploadFile): Promise<FormData> {
  const form = new FormData();
  const name = file.fileName ?? "photo.jpg";
  const type = file.mimeType ?? "image/jpeg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(file.uri)).blob();
    form.append("file", blob, name);
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor.
    form.append("file", { uri: file.uri, name, type } as unknown as Blob);
  }
  return form;
}

export const photoApi = {
  list: () => api<{ items: PhotoRef[] }>("/me/photos"),
  upload: async (file: UploadFile) => api<PhotoRef>("/me/photos", { method: "POST", form: await buildForm(file), timeoutMs: 60_000 }),
  replace: async (id: string, file: UploadFile) =>
    api<PhotoRef>(`/me/photos/${id}`, { method: "PUT", form: await buildForm(file), timeoutMs: 60_000 }),
  remove: (id: string) => api(`/me/photos/${id}`, { method: "DELETE" }),
  setMain: (id: string) => api<{ items: PhotoRef[] }>(`/me/photos/${id}/main`, { method: "POST" }),
  reorder: (photoIds: string[]) => api<{ items: PhotoRef[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};

export const discoverApi = {
  deck: (limit = 10) => api<{ items: Card[] }>(`/discover?limit=${limit}`),
  like: (userId: string) => api<SwipeResult>(`/discover/${userId}/like`, { method: "POST" }),
  pass: (userId: string) => api<SwipeResult>(`/discover/${userId}/pass`, { method: "POST" }),
  profile: (userId: string) => api<Card>(`/users/${userId}`)
};

export const chatApi = {
  conversations: (cursor?: string) => api<Page<Conversation>>(`/matches?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  messages: (matchId: string, before?: number) =>
    api<Page<Message>>(`/matches/${matchId}/messages?limit=30${before ? `&before=${before}` : ""}`),
  send: (matchId: string, body: string) => api<Message>(`/matches/${matchId}/messages`, { method: "POST", body: { body } }),
  markRead: (matchId: string) => api(`/matches/${matchId}/read`, { method: "POST" }),
  hide: (matchId: string) => api(`/matches/${matchId}/hide`, { method: "POST" }),
  unmatch: (matchId: string) => api(`/matches/${matchId}`, { method: "DELETE" })
};

export const safetyApi = {
  block: (userId: string) => api(`/users/${userId}/block`, { method: "POST" }),
  unblock: (userId: string) => api(`/users/${userId}/block`, { method: "DELETE" }),
  blocked: () => api<{ items: { userId: string; firstName: string }[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string) =>
    api(`/users/${userId}/report`, { method: "POST", body: { reason, details } })
};

export const notificationApi = {
  list: (cursor?: string) =>
    api<Page<NotificationItem> & { unreadCount: number }>(
      `/notifications?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`
    ),
  markRead: (id: string) => api(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: () => api<{ ticket: string }>("/ws/ticket", { method: "POST" })
};
