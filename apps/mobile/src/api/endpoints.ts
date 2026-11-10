import { Platform } from "react-native";
import { api } from "./client";
import type {
  AppNotification,
  AuthResponse,
  AuthUser,
  BlockedUser,
  Interest,
  Match,
  MeProfile,
  Message,
  Page,
  Photo,
  Preferences,
  ProfileCard,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

// --- Auth -------------------------------------------------------------------
export const authApi = {
  register: (email: string, password: string) => api<AuthResponse>("/auth/register", { body: { email, password }, auth: false }),
  login: (email: string, password: string) => api<AuthResponse>("/auth/login", { body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api("/auth/logout", { body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api("/auth/password/forgot", { body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) => api("/auth/password/reset", { body: { email, code, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) => api("/me/password", { body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api("/me", { method: "DELETE", body: { password } })
};

// --- Profile ----------------------------------------------------------------
export const profileApi = {
  me: () => api<MeProfile>("/profile"),
  upsert: (input: ProfileInput) => api<MeProfile>("/profile", { method: "PUT", body: input }),
  updateLocation: (latitude: number, longitude: number, city?: string) => api("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  updatePreferences: (prefs: Preferences) => api("/profile/preferences", { method: "PUT", body: prefs }),
  updatePrivacy: (isDiscoverable: boolean, showDistance: boolean) => api("/profile/privacy", { method: "PUT", body: { isDiscoverable, showDistance } }),
  interests: async () => (await api<{ interests: Interest[] }>("/interests")).interests,
  view: (userId: string) => api<ProfileCard>(`/profiles/${userId}`)
};

// --- Photos -----------------------------------------------------------------
async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string) => (await api<{ photos: Photo[] }>("/profile/photos", { form: await photoForm(uri), timeoutMs: 60_000 })).photos,
  replace: async (photoId: string, uri: string) =>
    (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "PUT", form: await photoForm(uri), timeoutMs: 60_000 })).photos,
  remove: async (photoId: string) => (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) => (await api<{ photos: Photo[] }>("/profile/photos/order", { method: "PUT", body: { photoIds } })).photos
};

// --- Discovery ----------------------------------------------------------------
export const discoveryApi = {
  deck: async (exclude: string[], limit = 10) => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (exclude.length) params.set("exclude", exclude.slice(-60).join(","));
    return (await api<{ profiles: ProfileCard[] }>(`/discovery?${params.toString()}`)).profiles;
  },
  swipe: (targetId: string, action: SwipeAction) => api<SwipeResult>("/discovery/swipes", { body: { targetId, action } })
};

// --- Matches & chat --------------------------------------------------------------
function withCursor(path: string, cursor?: string | null, limit?: number) {
  const params = new URLSearchParams();
  if (cursor) params.set("cursor", cursor);
  if (limit) params.set("limit", String(limit));
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

export const matchesApi = {
  list: async (cursor?: string | null): Promise<Page<Match>> => {
    const res = await api<{ matches: Match[]; nextCursor: string | null }>(withCursor("/matches", cursor, 20));
    return { items: res.matches, nextCursor: res.nextCursor };
  },
  get: (matchId: string) => api<Match>(`/matches/${matchId}`),
  messages: async (matchId: string, cursor?: string | null): Promise<Page<Message>> => {
    const res = await api<{ messages: Message[]; nextCursor: string | null }>(withCursor(`/matches/${matchId}/messages`, cursor, 30));
    return { items: res.messages, nextCursor: res.nextCursor };
  },
  send: (matchId: string, body: string) => api<Message>(`/matches/${matchId}/messages`, { body: { body } }),
  markRead: (matchId: string) => api(`/matches/${matchId}/read`, { method: "POST" }),
  unmatch: (matchId: string) => api(`/matches/${matchId}`, { method: "DELETE" })
};

// --- Notifications ------------------------------------------------------------
export const notificationsApi = {
  list: async (cursor?: string | null) => {
    const res = await api<{ notifications: AppNotification[]; unreadCount: number; nextCursor: string | null }>(withCursor("/notifications", cursor, 30));
    return { items: res.notifications, nextCursor: res.nextCursor, unreadCount: res.unreadCount };
  },
  unreadCount: async () => (await api<{ unreadCount: number }>("/notifications/unread-count")).unreadCount,
  markRead: (id: string) => api(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};

// --- Safety --------------------------------------------------------------------
export const safetyApi = {
  blocked: async () => (await api<{ blocked: BlockedUser[] }>("/blocks")).blocked,
  block: (userId: string) => api("/blocks", { body: { userId } }),
  unblock: (userId: string) => api(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) => api("/reports", { body: { userId, reason, details } })
};

// --- Realtime --------------------------------------------------------------------
export const realtimeApi = {
  ticket: async () => (await api<{ ticket: string }>("/realtime/ticket", { method: "POST" })).ticket
};
