import { Platform } from "react-native";
import { api } from "./client";
import type {
  AppNotification,
  AuthResponse,
  AuthUser,
  BlockedUser,
  ChatSummary,
  Interest,
  Match,
  Message,
  MessagesPage,
  MyProfile,
  Photo,
  Preferences,
  ProfileCard,
  ProfileInput,
  ReportReason,
  SwipeDecision,
  SwipeResult
} from "./types";

// --- Auth & account -------------------------------------------------------

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => api<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, password: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api<void>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};

// --- Own profile ----------------------------------------------------------

export const profileApi = {
  me: () => api<MyProfile>("/me/profile"),
  save: (input: ProfileInput) => api<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => api<MyProfile>("/me/preferences", { method: "PUT", body: prefs }),
  savePrivacy: (privacy: { isDiscoverable?: boolean; showDistance?: boolean }) =>
    api<MyProfile>("/me/privacy", { method: "PUT", body: privacy }),
  saveLocation: (latitude: number, longitude: number, city?: string) =>
    api<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city: city ?? "" } }),
  clearLocation: () => api<MyProfile>("/me/location", { method: "DELETE" }),
  saveInterests: (slugs: string[]) => api<MyProfile>("/me/interests", { method: "PUT", body: { slugs } }),
  interests: async () => (await api<{ interests: Interest[] }>("/interests")).interests
};

// --- Photos ---------------------------------------------------------------

async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native’s FormData accepts file descriptors with a local URI.
    const file = { uri, name: "photo.jpg", type: "image/jpeg" };
    form.append("photo", file as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string) =>
    (await api<{ photos: Photo[] }>("/me/photos", { method: "POST", form: await photoForm(uri), timeoutMs: 60_000 })).photos,
  replace: async (photoId: string, uri: string) =>
    (await api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(uri), timeoutMs: 60_000 })).photos,
  remove: async (photoId: string) => (await api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) => (await api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })).photos
};

// --- Discovery & matching -------------------------------------------------

export const discoveryApi = {
  feed: async (limit = 10) => (await api<{ profiles: ProfileCard[] }>(`/discovery?limit=${limit}`)).profiles,
  profile: (userId: string) => api<ProfileCard>(`/profiles/${userId}`),
  swipe: (targetUserId: string, decision: SwipeDecision) =>
    api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, decision } }),
  matches: async () => (await api<{ matches: Match[] }>("/matches?limit=100")).matches,
  unmatch: (userId: string) => api<void>(`/matches/${userId}`, { method: "DELETE" })
};

// --- Chat -----------------------------------------------------------------

export const chatApi = {
  list: async () => (await api<{ chats: ChatSummary[] }>("/chats?limit=50")).chats,
  messages: (userId: string, before?: string) =>
    api<MessagesPage>(`/chats/${userId}/messages?limit=30${before ? `&before=${before}` : ""}`),
  send: (userId: string, content: string) => api<Message>(`/chats/${userId}/messages`, { method: "POST", body: { content } }),
  markRead: (userId: string) => api<{ readAt: string }>(`/chats/${userId}/read`, { method: "POST" })
};

// --- Safety ---------------------------------------------------------------

export const safetyApi = {
  blocks: async () => (await api<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    api<void>("/reports", { method: "POST", body: { userId, reason, details } })
};

// --- Notifications & realtime ---------------------------------------------

export const notificationsApi = {
  list: () => api<{ notifications: AppNotification[]; unreadCount: number }>("/notifications?limit=50"),
  unreadCount: async () => (await api<{ unreadCount: number }>("/notifications/unread-count")).unreadCount,
  markRead: (id: string) => api<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: async () => (await api<{ ticket: string; expiresIn: number }>("/realtime/ticket", { method: "POST" })).ticket
};
