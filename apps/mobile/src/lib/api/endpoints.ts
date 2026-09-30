import { Platform } from "react-native";
import { api } from "./client";
import type {
  AuthSession,
  AuthUser,
  BlockedUser,
  ConversationsPage,
  Conversation,
  Gender,
  Interest,
  Message,
  ModerationReport,
  MessagesPage,
  NotificationsPage,
  OwnProfile,
  Photo,
  PublicProfile,
  RelationshipGoal,
  ReportReason,
  SwipeResult
} from "./types";

// Auth ----------------------------------------------------------------------
export const authApi = {
  register: (email: string, password: string) =>
    api<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    api<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => api<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => api<AuthUser>("/me"),
  forgotPassword: (email: string) => api<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    api<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  deleteAccount: (password: string) => api<void>("/me", { method: "DELETE", body: { password } })
};

// Profile -------------------------------------------------------------------
export type ProfilePatch = Partial<{
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  relationshipGoal: RelationshipGoal | "";
  city: string;
  discoverable: boolean;
  showDistance: boolean;
}>;

export const profileApi = {
  get: () => api<OwnProfile>("/profile"),
  update: (patch: ProfilePatch) => api<OwnProfile>("/profile", { method: "PATCH", body: patch }),
  updatePreferences: (prefs: OwnProfile["preferences"]) => api<OwnProfile>("/profile/preferences", { method: "PUT", body: prefs }),
  updateInterests: (interestIds: number[]) => api<OwnProfile>("/profile/interests", { method: "PUT", body: { interestIds } }),
  updateLocation: (latitude: number, longitude: number, city?: string) =>
    api<OwnProfile>("/profile/location", { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => api<OwnProfile>("/profile/location", { method: "DELETE" }),
  interests: () => api<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  publicProfile: (userId: string) => api<PublicProfile>(`/profiles/${userId}`)
};

// Photos --------------------------------------------------------------------
async function photoForm(uri: string, mimeType?: string | null): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors.
    form.append("photo", { uri, name: "photo.jpg", type: mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (uri: string, mimeType?: string | null) =>
    (await api<{ photos: Photo[] }>("/profile/photos", { method: "POST", form: await photoForm(uri, mimeType), timeoutMs: 60_000 })).photos,
  replace: async (photoId: string, uri: string, mimeType?: string | null) =>
    (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "PUT", form: await photoForm(uri, mimeType), timeoutMs: 60_000 }))
      .photos,
  remove: async (photoId: string) => (await api<{ photos: Photo[] }>(`/profile/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) => (await api<{ photos: Photo[] }>("/profile/photos/order", { method: "PUT", body: { photoIds } })).photos
};

// Discovery & matching ------------------------------------------------------
export const discoveryApi = {
  next: (limit = 10) => api<{ profiles: PublicProfile[] }>(`/discovery?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetUserId: string, action: "like" | "pass") => api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};

// Chat ----------------------------------------------------------------------
export const chatApi = {
  conversations: (cursor?: string | null) =>
    api<ConversationsPage>(`/conversations?limit=20${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  conversation: (id: string) => api<Conversation>(`/conversations/${id}`),
  messages: (id: string, cursor?: string | null) =>
    api<MessagesPage>(`/conversations/${id}/messages?limit=30${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`),
  send: (id: string, body: string) => api<Message>(`/conversations/${id}/messages`, { method: "POST", body: { body } }),
  markRead: (id: string) => api<void>(`/conversations/${id}/read`, { method: "POST" }),
  hide: (id: string) => api<void>(`/conversations/${id}`, { method: "DELETE" })
};

// Notifications -------------------------------------------------------------
export const notificationsApi = {
  registerPushToken: (token: string, platform: "ios" | "android") =>
    api<void>("/push-tokens", { method: "PUT", body: { token, platform } }),
  unregisterPushToken: (token: string) => api<void>("/push-tokens", { method: "DELETE", body: { token } }),
  list: (offset = 0) => api<NotificationsPage>(`/notifications?limit=20&offset=${offset}`),
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};

// Safety --------------------------------------------------------------------
export const safetyApi = {
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => api<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string) =>
    api<void>("/reports", { method: "POST", body: { userId, reason, details, block: true } })
};

// Realtime ------------------------------------------------------------------
export const realtimeApi = {
  ticket: () => api<{ ticket: string }>("/realtime/ticket", { method: "POST" }).then((r) => r.ticket)
};

// Moderation (moderators only) --------------------------------------------
export const moderationApi = {
  reports: (offset = 0) =>
    api<{ reports: ModerationReport[]; nextOffset?: number }>(`/moderation/reports?status=open&limit=20&offset=${offset}`),
  resolve: (reportId: string, status: "reviewed" | "dismissed") =>
    api<void>(`/moderation/reports/${reportId}/resolve`, { method: "POST", body: { status } }),
  suspend: (userId: string) => api<void>(`/moderation/users/${userId}/suspend`, { method: "POST" }),
  unsuspend: (userId: string) => api<void>(`/moderation/users/${userId}/unsuspend`, { method: "POST" })
};
