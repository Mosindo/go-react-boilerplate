import { Platform } from "react-native";
import { api } from "./client";
import type {
  AuthSession,
  AuthUser,
  BlockedUser,
  Card,
  ConversationsPage,
  Interest,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Preferences,
  ProfileInput,
  ReportReason,
  SwipeDirection,
  SwipeResult
} from "./types";

// --- auth ---------------------------------------------------------------

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  refresh: (refreshToken: string) =>
    api<AuthSession>("/auth/refresh", { method: "POST", body: { refreshToken }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) =>
    api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  me: () => api<AuthUser>("/me"),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};

// --- profile ------------------------------------------------------------

export type LocalImage = { uri: string; mimeType: string; fileName: string };

async function photoForm(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, image.fileName);
  } else {
    // React Native's FormData accepts file descriptors in this shape.
    form.append("photo", { uri: image.uri, name: image.fileName, type: image.mimeType } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  mine: () => api<MyProfile>("/me/profile"),
  interests: () => api<{ interests: Interest[] }>("/interests"),
  saveProfile: (input: ProfileInput) => api<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (input: Preferences) => api<MyProfile>("/me/preferences", { method: "PUT", body: input }),
  saveInterests: (slugs: string[]) => api<MyProfile>("/me/interests", { method: "PUT", body: { interests: slugs } }),
  setLocation: (latitude: number, longitude: number) =>
    api<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => api<MyProfile>("/me/location", { method: "DELETE" }),
  setVisibility: (input: { discoverable?: boolean; showDistance?: boolean }) =>
    api<MyProfile>("/me/visibility", { method: "PUT", body: input }),
  uploadPhoto: async (image: LocalImage) =>
    api<MyProfile>("/me/photos", { method: "POST", form: await photoForm(image), timeoutMs: 60_000 }),
  replacePhoto: async (photoId: string, image: LocalImage) =>
    api<MyProfile>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(image), timeoutMs: 60_000 }),
  deletePhoto: (photoId: string) => api<MyProfile>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorderPhotos: (photoIds: string[]) => api<MyProfile>("/me/photos/order", { method: "PUT", body: { photoIds } }),
  view: (userId: string) => api<Card>(`/profiles/${userId}`)
};

// --- discovery & matching -------------------------------------------------

export const discoveryApi = {
  feed: (limit = 10) => api<{ profiles: Card[] }>(`/discover?limit=${limit}`),
  swipe: (targetUserId: string, direction: SwipeDirection) =>
    api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, direction } }),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};

// --- chat ---------------------------------------------------------------

export const chatApi = {
  conversations: (cursor?: string) =>
    api<ConversationsPage>(`/conversations?limit=20${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  messages: (matchId: string, cursor?: string) =>
    api<MessagesPage>(
      `/conversations/${matchId}/messages?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`
    ),
  send: (matchId: string, body: string) =>
    api<Message>(`/conversations/${matchId}/messages`, { method: "POST", body: { body } }),
  markRead: (matchId: string) => api<void>(`/conversations/${matchId}/read`, { method: "POST" }),
  hide: (matchId: string) => api<void>(`/conversations/${matchId}`, { method: "DELETE" })
};

// --- safety ---------------------------------------------------------------

export const safetyApi = {
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks"),
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    api<{ id: string }>("/reports", { method: "POST", body: { userId, reason, details } })
};

// --- notifications & realtime --------------------------------------------

export const notificationsApi = {
  list: (offset = 0) => api<NotificationsPage>(`/notifications?limit=20&offset=${offset}`),
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: () => api<{ ticket: string; expiresIn: number }>("/realtime/ticket", { method: "POST" })
};
