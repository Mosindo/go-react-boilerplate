import { api } from "./client";
import type {
  AuthSession,
  BlockedUser,
  ConversationsPage,
  Gender,
  Interest,
  Match,
  MatchesPage,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  PhotoRef,
  Preferences,
  ProfileCard,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

export type UploadFile = { uri: string; name: string; type: string };

async function photoForm(file: UploadFile): Promise<FormData> {
  const form = new FormData();
  if (file.uri.startsWith("blob:") || file.uri.startsWith("data:") || typeof document !== "undefined") {
    // Web: expo-image-picker returns a blob/data URI that must be turned into a real Blob.
    const blob = await (await fetch(file.uri)).blob();
    form.append("photo", blob, file.name);
  } else {
    // Native: React Native's FormData accepts a { uri, name, type } descriptor.
    form.append("photo", { uri: file.uri, name: file.name, type: file.type } as unknown as Blob);
  }
  return form;
}

export const authApi = {
  register: (input: { email: string; password: string; firstName: string; birthDate: string }) =>
    api<AuthSession>("/auth/register", { method: "POST", body: input, auth: false }),
  login: (email: string, password: string) =>
    api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    api("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) =>
    api("/auth/forgot-password", { method: "POST", body: { email }, auth: false }),
  resetPassword: (code: string, newPassword: string) =>
    api("/auth/reset-password", { method: "POST", body: { code, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api("/auth/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  me: () => api<MyProfile>("/profile"),
  update: (
    patch: Partial<{
      firstName: string;
      gender: Gender;
      bio: string;
      city: string;
      interests: string[];
      discoverable: boolean;
      showDistance: boolean;
    }>
  ) => api<MyProfile>("/profile", { method: "PUT", body: patch }),
  setLocation: (latitude: number, longitude: number, city?: string) =>
    api<MyProfile>("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  get: (userId: string) => api<ProfileCard>(`/profiles/${userId}`),
  preferences: () => api<Preferences>("/preferences"),
  updatePreferences: (p: Preferences) => api<Preferences>("/preferences", { method: "PUT", body: p }),
  interests: async () => (await api<{ interests: Interest[] }>("/interests")).interests
};

export const photosApi = {
  list: async () => (await api<{ photos: PhotoRef[] }>("/photos")).photos,
  upload: async (file: UploadFile) =>
    api<PhotoRef>("/photos", { method: "POST", form: await photoForm(file) }),
  replace: async (id: string, file: UploadFile) =>
    api<PhotoRef>(`/photos/${id}`, { method: "PUT", form: await photoForm(file) }),
  remove: (id: string) => api(`/photos/${id}`, { method: "DELETE" }),
  reorder: async (ids: string[]) =>
    (await api<{ photos: PhotoRef[] }>("/photos/order", { method: "PUT", body: { ids } })).photos
};

export const discoverApi = {
  next: async (limit = 10) =>
    (await api<{ profiles: ProfileCard[] }>("/discover", { query: { limit } })).profiles,
  swipe: (userId: string, action: SwipeAction) =>
    api<SwipeResult>("/swipes", { method: "POST", body: { userId, action } })
};

export const matchesApi = {
  list: (before?: string) => api<MatchesPage>("/matches", { query: { limit: 30, before } }),
  unmatch: (matchId: string) => api(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: (before?: string) =>
    api<ConversationsPage>("/conversations", { query: { limit: 30, before } }),
  messages: (conversationId: string, before?: string) =>
    api<MessagesPage>(`/conversations/${conversationId}/messages`, { query: { limit: 30, before } }),
  send: (conversationId: string, body: string) =>
    api<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => api(`/conversations/${conversationId}/read`, { method: "POST" }),
  clear: (conversationId: string) => api(`/conversations/${conversationId}`, { method: "DELETE" }),
  unreadCount: async () => (await api<{ unreadCount: number }>("/conversations/unread-count")).unreadCount
};

export const notificationsApi = {
  list: (before?: string) => api<NotificationsPage>("/notifications", { query: { limit: 20, before } }),
  markRead: (id: string) => api(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => api("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: async () => (await api<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  report: (userId: string, reason: ReportReason, details: string) =>
    api("/reports", { method: "POST", body: { userId, reason, details } })
};

export type { Match };
