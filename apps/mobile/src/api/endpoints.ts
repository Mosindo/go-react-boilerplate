import { Platform } from "react-native";
import { apiRequest } from "./client";
import type {
  AppNotification,
  AuthUser,
  BlockedUser,
  Conversation,
  Interest,
  Message,
  OwnProfile,
  Photo,
  Preferences,
  PublicProfile,
  RealtimeEvent,
  ReportReason,
  SaveProfileInput,
  Session,
  SwipeAction,
  SwipeResult
} from "./types";

type AuthResponse = { accessToken: string; refreshToken: string; user: AuthUser };

function toSession(response: AuthResponse): Session {
  return { accessToken: response.accessToken, refreshToken: response.refreshToken, user: response.user };
}

export const authApi = {
  register: async (email: string, password: string) =>
    toSession(
      await apiRequest<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, anonymous: true })
    ),
  login: async (email: string, password: string) =>
    toSession(
      await apiRequest<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, anonymous: true })
    ),
  logout: (refreshToken: string) =>
    apiRequest("/auth/logout", { method: "POST", body: { refreshToken }, anonymous: true }),
  me: () => apiRequest<AuthUser>("/me"),
  forgotPassword: (email: string) => apiRequest("/auth/forgot", { method: "POST", body: { email }, anonymous: true }),
  resetPassword: (token: string, password: string) =>
    apiRequest("/auth/reset", { method: "POST", body: { token, password }, anonymous: true }),
  changePassword: (currentPassword: string, newPassword: string) =>
    apiRequest("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => apiRequest("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  get: () => apiRequest<OwnProfile>("/profile"),
  save: (input: SaveProfileInput) => apiRequest<OwnProfile>("/profile", { method: "PUT", body: input }),
  preferences: () => apiRequest<Preferences>("/preferences"),
  savePreferences: (prefs: Preferences) => apiRequest<Preferences>("/preferences", { method: "PUT", body: prefs }),
  interests: async () => (await apiRequest<{ interests: Interest[] }>("/interests")).interests,
  publicProfile: (userId: string) => apiRequest<PublicProfile>(`/profiles/${userId}`)
};

export type PickedImage = { uri: string; mimeType?: string | null; fileName?: string | null };

async function toFormData(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  const name = image.fileName ?? "photo.jpg";
  const type = image.mimeType ?? "image/jpeg";
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("file", blob, name);
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor.
    form.append("file", { uri: image.uri, name, type } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  list: async () => (await apiRequest<{ photos: Photo[] }>("/photos")).photos,
  upload: async (image: PickedImage) =>
    (
      await apiRequest<{ photos: Photo[] }>("/photos", {
        method: "POST",
        form: await toFormData(image),
        timeoutMs: 60_000
      })
    ).photos,
  replace: async (photoId: string, image: PickedImage) =>
    (
      await apiRequest<{ photos: Photo[] }>(`/photos/${photoId}`, {
        method: "PUT",
        form: await toFormData(image),
        timeoutMs: 60_000
      })
    ).photos,
  remove: async (photoId: string) =>
    (await apiRequest<{ photos: Photo[] }>(`/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) =>
    (await apiRequest<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } })).photos
};

export const matchingApi = {
  discover: async (limit = 10) =>
    (await apiRequest<{ profiles: PublicProfile[] }>(`/discover?limit=${limit}`)).profiles,
  swipe: (targetUserId: string, action: SwipeAction) =>
    apiRequest<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (matchId: string) => apiRequest(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: async () =>
    (await apiRequest<{ conversations: Conversation[] }>("/conversations?limit=100")).conversations,
  messages: (conversationId: string, before?: string) =>
    apiRequest<{ messages: Message[]; nextBefore?: string }>(
      `/conversations/${conversationId}/messages?limit=30${before ? `&before=${encodeURIComponent(before)}` : ""}`
    ),
  send: (conversationId: string, body: string) =>
    apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => apiRequest(`/conversations/${conversationId}/read`, { method: "POST" }),
  hide: (conversationId: string) => apiRequest(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const safetyApi = {
  block: (userId: string) => apiRequest("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: async () => (await apiRequest<{ blocked: BlockedUser[] }>("/blocks")).blocked,
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    apiRequest("/reports", { method: "POST", body: { userId, reason, details, block } })
};

export const notificationsApi = {
  list: () => apiRequest<{ notifications: AppNotification[]; unreadCount: number }>("/notifications?limit=50"),
  markRead: (id: string) => apiRequest(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest("/notifications/read-all", { method: "POST" })
};

export const eventsApi = {
  cursor: () => apiRequest<{ cursor: number }>("/events"),
  poll: (after: number, signal: AbortSignal) =>
    apiRequest<{ cursor: number; events: RealtimeEvent[] }>(`/events?after=${after}`, { signal, timeoutMs: 40_000 })
};
