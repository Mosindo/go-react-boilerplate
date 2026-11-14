import { Platform } from "react-native";
import { api } from "./client";
import type {
  AuthSession,
  AuthUser,
  BlockedUser,
  Conversation,
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

// --- auth -------------------------------------------------------------
export const authApi = {
  register: (email: string, password: string) =>
    api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, password: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};

// --- profile ----------------------------------------------------------
export const profileApi = {
  interests: () => api<{ interests: Interest[] }>("/interests", { auth: false }).then((r) => r.interests),
  mine: () => api<MyProfile>("/me/profile"),
  update: (input: ProfileInput) => api<MyProfile>("/me/profile", { method: "PUT", body: input }),
  updatePreferences: (prefs: Preferences) => api<Preferences>("/me/preferences", { method: "PUT", body: prefs }),
  updateLocation: (latitude: number, longitude: number, city: string) =>
    api<void>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => api<void>("/me/location", { method: "DELETE" }),
  updatePrivacy: (privacy: { discoverable?: boolean; showDistance?: boolean }) =>
    api<void>("/me/privacy", { method: "PUT", body: privacy }),
  view: (userId: string) => api<PublicProfile>(`/profiles/${userId}`)
};

// --- photos -----------------------------------------------------------
async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a { uri, name, type } file descriptor.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string) =>
    api<Photo>("/me/photos", { method: "POST", form: await photoForm(uri), timeoutMs: 60_000 }),
  replace: async (photoId: string, uri: string) =>
    api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(uri), timeoutMs: 60_000 }),
  remove: (photoId: string) => api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};

// --- discovery & matching ---------------------------------------------
export const matchingApi = {
  discover: (limit = 10) => api<{ profiles: PublicProfile[] }>(`/discovery?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetUserId: string, action: SwipeAction) =>
    api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};

// --- chat -------------------------------------------------------------
export const chatApi = {
  conversations: (limit = 50, offset = 0) =>
    api<{ conversations: Conversation[] }>(`/conversations?limit=${limit}&offset=${offset}`).then((r) => r.conversations),
  messages: (matchId: string, before?: string) =>
    api<MessagesPage>(`/conversations/${matchId}/messages?limit=30${before ? `&before=${before}` : ""}`),
  send: (matchId: string, body: string) => api<Message>(`/conversations/${matchId}/messages`, { method: "POST", body: { body } }),
  markRead: (matchId: string) => api<{ readAt: string }>(`/conversations/${matchId}/read`, { method: "POST" }),
  clear: (matchId: string) => api<void>(`/conversations/${matchId}/messages`, { method: "DELETE" })
};

// --- notifications ----------------------------------------------------
export const notificationsApi = {
  list: (limit = 30, offset = 0) => api<NotificationsPage>(`/notifications?limit=${limit}&offset=${offset}`),
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};

// --- safety -----------------------------------------------------------
export const safetyApi = {
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    api<void>("/reports", { method: "POST", body: { userId, reason, details } })
};
