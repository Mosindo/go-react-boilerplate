import { Platform } from "react-native";
import { apiRequest } from "./client";
import type {
  AppNotification,
  AuthResponse,
  AuthUser,
  BlockedUser,
  ChatMessage,
  ChatSummary,
  FeedItem,
  Interest,
  MeProfile,
  MessagesPage,
  Photo,
  Preferences,
  ProfileCard,
  ReportReason,
  SwipeAction,
  SwipeResult,
  UpsertProfileInput
} from "./types";

// --- auth ---
export const authApi = {
  register: (email: string, password: string) =>
    apiRequest<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    apiRequest<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => apiRequest<AuthUser>("/me"),
  forgotPassword: (email: string) => apiRequest("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    apiRequest("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  deleteAccount: (password: string) => apiRequest("/me", { method: "DELETE", body: { password } })
};

// --- profile ---
export const profileApi = {
  me: () => apiRequest<MeProfile>("/me/profile"),
  upsert: (input: UpsertProfileInput) => apiRequest<MeProfile>("/me/profile", { method: "PUT", body: input }),
  preferences: (prefs: Preferences) => apiRequest<MeProfile>("/me/preferences", { method: "PUT", body: prefs }),
  interests: (interestIds: number[]) => apiRequest<MeProfile>("/me/interests", { method: "PUT", body: { interestIds } }),
  settings: (settings: { isVisible?: boolean; showDistance?: boolean }) =>
    apiRequest<MeProfile>("/me/settings", { method: "PATCH", body: settings }),
  setLocation: (latitude: number, longitude: number) => apiRequest("/me/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => apiRequest("/me/location", { method: "DELETE" }),
  catalogue: () => apiRequest<{ interests: Interest[] }>("/interests"),
  view: (userId: string) => apiRequest<ProfileCard>(`/profiles/${userId}`)
};

// --- photos ---
async function photoForm(uri: string, mimeType?: string | null): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors with uri/name/type.
    form.append("photo", { uri, name: "photo.jpg", type: mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string, mimeType?: string | null) =>
    apiRequest<{ photos: Photo[] }>("/me/photos", { method: "POST", body: await photoForm(uri, mimeType), timeoutMs: 60_000 }),
  replace: async (photoId: string, uri: string, mimeType?: string | null) =>
    apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, {
      method: "PUT",
      body: await photoForm(uri, mimeType),
      timeoutMs: 60_000
    }),
  remove: (photoId: string) => apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => apiRequest<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};

// --- discovery & matching ---
export const discoveryApi = {
  feed: (exclude: string[], limit = 10) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (exclude.length > 0) {
      params.set("exclude", exclude.join(","));
    }
    return apiRequest<{ profiles: FeedItem[] }>(`/discovery?${params.toString()}`);
  },
  swipe: (targetUserId: string, action: SwipeAction) =>
    apiRequest<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (userId: string) => apiRequest(`/matches/${userId}`, { method: "DELETE" })
};

// --- chat ---
export const chatApi = {
  list: (offset: number, limit = 20) => apiRequest<{ chats: ChatSummary[] }>(`/chats?limit=${limit}&offset=${offset}`),
  messages: (userId: string, before?: string) =>
    apiRequest<MessagesPage>(`/chats/${userId}/messages?limit=40${before ? `&before=${before}` : ""}`),
  send: (userId: string, content: string) => apiRequest<ChatMessage>(`/chats/${userId}/messages`, { method: "POST", body: { content } }),
  markRead: (userId: string) => apiRequest<{ readAt: string }>(`/chats/${userId}/read`, { method: "POST" }),
  clear: (userId: string) => apiRequest(`/chats/${userId}/messages`, { method: "DELETE" })
};

// --- notifications ---
export const notificationsApi = {
  list: (offset: number, limit = 30) =>
    apiRequest<{ notifications: AppNotification[]; unreadCount: number }>(`/notifications?limit=${limit}&offset=${offset}`),
  markRead: (id: string) => apiRequest<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest("/notifications/read-all", { method: "POST" })
};

// --- safety ---
export const safetyApi = {
  block: (userId: string) => apiRequest("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => apiRequest<{ blocks: BlockedUser[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string) =>
    apiRequest<{ id: string; blocked: boolean }>("/reports", { method: "POST", body: { userId, reason, details } })
};
