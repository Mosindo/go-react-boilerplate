import { Platform } from "react-native";
import { apiRequest } from "./client";
import type {
  AppNotification,
  AuthSession,
  AuthUser,
  BlockedUser,
  Conversation,
  Interest,
  MessagesPage,
  Message,
  MyProfile,
  NotificationsPage,
  PhotoRef,
  Preferences,
  ProfileCard,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResponse
} from "./types";

export const authApi = {
  register: (email: string, password: string) => apiRequest<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) => apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => apiRequest<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => apiRequest<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, password: string) =>
    apiRequest<AuthSession>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false }),
  me: () => apiRequest<AuthUser>("/me"),
  changePassword: (currentPassword: string, newPassword: string) =>
    apiRequest<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => apiRequest<void>("/me", { method: "DELETE", body: { password } }),
  realtimeTicket: () => apiRequest<{ ticket: string; expiresIn: number }>("/realtime/ticket", { method: "POST" })
};

export const profileApi = {
  interests: () => apiRequest<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  mine: () => apiRequest<MyProfile>("/profile"),
  save: (input: ProfileInput) => apiRequest<MyProfile>("/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => apiRequest<Preferences>("/profile/preferences", { method: "PUT", body: prefs }),
  setLocation: (latitude: number, longitude: number) => apiRequest<void>("/profile/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => apiRequest<void>("/profile/location", { method: "DELETE" }),
  setDiscoverable: (discoverable: boolean) => apiRequest<void>("/profile/visibility", { method: "PUT", body: { discoverable } }),
  view: (userId: string) => apiRequest<ProfileCard>(`/profiles/${userId}`)
};

export type LocalImage = { uri: string; mimeType?: string | null; fileName?: string | null };

async function photoForm(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  const name = image.fileName ?? "photo.jpg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, name);
  } else {
    // React Native's FormData accepts a file descriptor object.
    form.append("photo", { uri: image.uri, name, type: image.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (image: LocalImage) => apiRequest<PhotoRef>("/profile/photos", { method: "POST", form: await photoForm(image), timeoutMs: 60_000 }),
  replace: async (photoId: string, image: LocalImage) =>
    apiRequest<PhotoRef>(`/profile/photos/${photoId}`, { method: "PUT", form: await photoForm(image), timeoutMs: 60_000 }),
  remove: (photoId: string) => apiRequest<void>(`/profile/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => apiRequest<{ photos: PhotoRef[] }>("/profile/photos/order", { method: "PUT", body: { photoIds } })
};

export const matchingApi = {
  discover: (limit = 10) => apiRequest<{ profiles: ProfileCard[] }>(`/discovery?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetUserId: string, action: SwipeAction) => apiRequest<SwipeResponse>("/swipes", { method: "POST", body: { targetUserId, action } }),
  likesReceived: (limit = 30, offset = 0) => apiRequest<{ profiles: ProfileCard[] }>(`/likes/received?limit=${limit}&offset=${offset}`).then((r) => r.profiles),
  unmatch: (matchId: string) => apiRequest<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  list: (filter: "all" | "new" | "active", limit = 30, offset = 0) =>
    apiRequest<{ conversations: Conversation[] }>(`/conversations?filter=${filter}&limit=${limit}&offset=${offset}`).then((r) => r.conversations),
  get: (conversationId: string) => apiRequest<Conversation>(`/conversations/${conversationId}`),
  messages: (conversationId: string, before?: string, limit = 30) =>
    apiRequest<MessagesPage>(`/conversations/${conversationId}/messages?limit=${limit}${before ? `&before=${before}` : ""}`),
  send: (conversationId: string, body: string) => apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => apiRequest<{ lastReadAt: string | null }>(`/conversations/${conversationId}/read`, { method: "POST" }),
  clear: (conversationId: string) => apiRequest<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const notificationsApi = {
  list: (limit = 30, offset = 0) => apiRequest<NotificationsPage>(`/notifications?limit=${limit}&offset=${offset}`),
  markRead: (id: string) => apiRequest<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest<void>("/notifications/read-all", { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => apiRequest<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => apiRequest<{ blocked: BlockedUser[] }>("/blocks").then((r) => r.blocked),
  report: (userId: string, reason: ReportReason, details: string) => apiRequest<void>("/reports", { method: "POST", body: { userId, reason, details } })
};

export type { AppNotification };
