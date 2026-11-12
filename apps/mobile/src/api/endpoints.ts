import { api } from "./client";
import type {
  AppNotification,
  AuthSession,
  AuthUser,
  BlockedUser,
  ChatSummary,
  Interest,
  Match,
  Message,
  MessagesPage,
  OwnProfile,
  Photo,
  Preferences,
  Privacy,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeResult
} from "./types";

// --- auth ---
export const authApi = {
  register: (email: string, password: string) => api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  refresh: (refreshToken: string) => api<AuthSession>("/auth/refresh", { method: "POST", body: { refreshToken }, auth: false }),
  logout: (refreshToken: string) => api("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => api("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  me: () => api<AuthUser>("/me"),
  changePassword: (currentPassword: string, newPassword: string) => api("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api("/me", { method: "DELETE", body: { password } })
};

// --- profile ---
export const profileApi = {
  get: () => api<OwnProfile>("/profile"),
  upsert: (input: ProfileInput) => api<OwnProfile>("/profile", { method: "PUT", body: input }),
  preferences: (prefs: Preferences) => api<OwnProfile>("/profile/preferences", { method: "PUT", body: prefs }),
  interests: (interestIds: number[]) => api<OwnProfile>("/profile/interests", { method: "PUT", body: { interestIds } }),
  location: (latitude: number, longitude: number, city?: string) =>
    api<OwnProfile>("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => api<OwnProfile>("/profile/location", { method: "DELETE" }),
  privacy: (p: Privacy) => api<OwnProfile>("/profile/privacy", { method: "PUT", body: p }),
  catalogue: async () => (await api<{ interests: Interest[] }>("/interests")).interests,
  publicProfile: (userId: string) => api<PublicProfile>(`/profiles/${userId}`)
};

export type LocalImage = { uri: string; mimeType?: string | null; fileName?: string | null; file?: File };

function photoForm(image: LocalImage): FormData {
  const form = new FormData();
  if (image.file) {
    // Web: the picker gives us a real File object.
    form.append("photo", image.file);
  } else {
    // Native: React Native's FormData accepts {uri, name, type}.
    form.append("photo", { uri: image.uri, name: image.fileName ?? "photo.jpg", type: image.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (image: LocalImage) => (await api<{ photos: Photo[] }>("/profile/photos", { method: "POST", form: photoForm(image) })).photos,
  replace: async (photoId: string, image: LocalImage) =>
    (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "PUT", form: photoForm(image) })).photos,
  remove: async (photoId: string) => (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) => (await api<{ photos: Photo[] }>("/profile/photos/order", { method: "PUT", body: { photoIds } })).photos
};

// --- discovery & matching ---
export const matchingApi = {
  discover: async (exclude: string[], limit = 10) => {
    const q = new URLSearchParams({ limit: String(limit) });
    if (exclude.length) q.set("exclude", exclude.join(","));
    return (await api<{ profiles: PublicProfile[] }>(`/discover?${q.toString()}`)).profiles;
  },
  swipe: (targetUserId: string, decision: "like" | "pass") => api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, decision } }),
  matches: async () => (await api<{ matches: Match[] }>("/matches?limit=100")).matches,
  unmatch: (matchId: string) => api(`/matches/${matchId}`, { method: "DELETE" })
};

// --- chat ---
export const chatApi = {
  list: async () => (await api<{ chats: ChatSummary[] }>("/chats?limit=50")).chats,
  unread: async () => (await api<{ unreadConversations: number }>("/chats/unread")).unreadConversations,
  messages: (userId: string, before?: string) => {
    const q = new URLSearchParams({ limit: "40" });
    if (before) q.set("before", before);
    return api<MessagesPage>(`/chats/${userId}/messages?${q.toString()}`);
  },
  send: (userId: string, content: string) => api<Message>(`/chats/${userId}/messages`, { method: "POST", body: { content } }),
  markRead: (userId: string) => api<{ readAt: string }>(`/chats/${userId}/read`, { method: "POST" }),
  hide: (userId: string) => api(`/chats/${userId}`, { method: "DELETE" })
};

// --- notifications ---
export const notificationsApi = {
  list: () => api<{ notifications: AppNotification[]; unreadCount: number }>("/notifications?limit=50"),
  unreadCount: async () => (await api<{ unreadCount: number }>("/notifications/unread-count")).unreadCount,
  markRead: (id: string) => api<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};

// --- safety ---
export const safetyApi = {
  block: (userId: string) => api("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: async () => (await api<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  report: (userId: string, reason: ReportReason, details: string) => api("/reports", { method: "POST", body: { userId, reason, details } })
};
