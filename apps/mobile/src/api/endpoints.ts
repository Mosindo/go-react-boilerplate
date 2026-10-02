import { request } from "./client";
import type {
  AuthResponse,
  BlockedUser,
  Card,
  Interest,
  MatchPage,
  Message,
  MessagePage,
  NotificationPage,
  Photo,
  Preferences,
  ProfileInput,
  ProfileStatus,
  ReportReason,
  Summary,
  SwipeResult,
  User,
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthResponse>("/auth/register", {
      method: "POST",
      body: { email, password },
      auth: false,
    }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", {
      method: "POST",
      body: { email, password },
      auth: false,
    }),
  logout: (refreshToken: string) =>
    request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) =>
    request<void>("/auth/forgot-password", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    request<void>("/auth/reset-password", {
      method: "POST",
      body: { email, code, newPassword },
      auth: false,
    }),
  me: () => request<User>("/me"),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) =>
    request<void>("/me", { method: "DELETE", body: { password } }),
};

export const profileApi = {
  get: () => request<ProfileStatus>("/profile"),
  save: (input: ProfileInput) =>
    request<ProfileStatus>("/profile", { method: "PUT", body: { ...input } }),
  savePreferences: (prefs: Preferences) =>
    request<ProfileStatus>("/preferences", { method: "PUT", body: { ...prefs } }),
  interests: () => request<{ interests: Interest[] }>("/interests"),
};

export interface UploadFile {
  uri: string;
  name: string;
  type: string;
}

async function toFormData(file: UploadFile): Promise<FormData> {
  const form = new FormData();
  if (typeof document !== "undefined") {
    // Web: React Native's {uri} convention does not exist, send a real Blob.
    const blob = await (await fetch(file.uri)).blob();
    form.append("file", blob, file.name);
  } else {
    // React Native serialises this object as a file part.
    form.append("file", file as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  list: () => request<{ photos: Photo[] }>("/photos"),
  upload: async (file: UploadFile) =>
    request<Photo>("/photos", { method: "POST", body: await toFormData(file) }),
  replace: async (id: string, file: UploadFile) =>
    request<Photo>(`/photos/${id}`, { method: "PUT", body: await toFormData(file) }),
  remove: (id: string) => request<void>(`/photos/${id}`, { method: "DELETE" }),
  makePrimary: (id: string) =>
    request<{ photos: Photo[] }>(`/photos/${id}/primary`, { method: "PUT" }),
  reorder: (ids: string[]) =>
    request<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { ids } }),
};

export const discoverApi = {
  list: (limit = 10) => request<{ profiles: Card[] }>(`/discover?limit=${limit}`),
  profile: (id: string) => request<Card>(`/profiles/${id}`),
  swipe: (targetId: string, action: "like" | "pass") =>
    request<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } }),
};

export const matchesApi = {
  list: (cursor?: string) =>
    request<MatchPage>(`/matches?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  unmatch: (id: string) => request<void>(`/matches/${id}`, { method: "DELETE" }),
  messages: (id: string, before?: number) =>
    request<MessagePage>(`/matches/${id}/messages?limit=40${before ? `&before=${before}` : ""}`),
  send: (id: string, body: string) =>
    request<Message>(`/matches/${id}/messages`, { method: "POST", body: { body } }),
  markRead: (id: string) => request<{ updated: number }>(`/matches/${id}/read`, { method: "POST" }),
  clear: (id: string) => request<void>(`/matches/${id}/messages`, { method: "DELETE" }),
};

export const notificationsApi = {
  list: (cursor?: string) =>
    request<NotificationPage>(
      `/notifications?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    ),
  summary: () => request<Summary>("/notifications/summary"),
  markAllRead: () => request<void>("/notifications/read", { method: "POST" }),
  markRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
};

export const safetyApi = {
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => request<{ blocks: BlockedUser[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details, block } }),
};

export const realtimeApi = {
  ticket: () => request<{ ticket: string }>("/realtime/ticket", { method: "POST" }),
};
