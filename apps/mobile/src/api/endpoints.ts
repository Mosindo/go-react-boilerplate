import { apiRequest } from "./client";
import type {
  AppNotification,
  AuthResponse,
  AuthUser,
  BlockedUser,
  Conversation,
  DiscoverCard,
  Gender,
  Interest,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Page,
  Photo,
  Preferences,
  PrivacySettings,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResponse
} from "./types";

function query(params: Record<string, string | number | null | undefined>): string {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== null && v !== undefined && v !== "")
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  return parts.length ? `?${parts.join("&")}` : "";
}

export const authApi = {
  register: (email: string, password: string) =>
    apiRequest<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    apiRequest<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => apiRequest<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => apiRequest<AuthUser>("/me"),
  forgotPassword: (email: string) => apiRequest<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    apiRequest<void>("/auth/password/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    apiRequest<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => apiRequest<void>("/me", { method: "DELETE", body: { password } })
};

export type ProfilePatch = Partial<{
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  interestIds: number[];
}>;

export const profileApi = {
  me: () => apiRequest<MyProfile>("/me/profile"),
  update: (patch: ProfilePatch) => apiRequest<MyProfile>("/me/profile", { method: "PATCH", body: patch }),
  updateLocation: (latitude: number, longitude: number, city?: string) =>
    apiRequest<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  updatePreferences: (prefs: Preferences) => apiRequest<MyProfile>("/me/preferences", { method: "PUT", body: prefs }),
  updateSettings: (settings: Partial<PrivacySettings>) => apiRequest<MyProfile>("/me/settings", { method: "PATCH", body: settings }),
  interests: () => apiRequest<{ interests: Interest[] }>("/interests"),
  view: (userId: string) => apiRequest<PublicProfile>(`/profiles/${userId}`)
};

export type UploadablePhoto = { uri: string; file?: Blob };

function photoForm(photo: UploadablePhoto): FormData {
  const form = new FormData();
  if (photo.file) {
    form.append("photo", photo.file, "photo.jpg");
  } else {
    // React Native's FormData accepts this file descriptor shape.
    form.append("photo", { uri: photo.uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: (photo: UploadablePhoto) =>
    apiRequest<{ photos: Photo[] }>("/me/photos", { method: "POST", form: photoForm(photo), timeoutMs: 60_000 }),
  replace: (photoId: string, photo: UploadablePhoto) =>
    apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "PUT", form: photoForm(photo), timeoutMs: 60_000 }),
  remove: (photoId: string) => apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => apiRequest<{ photos: Photo[] }>("/me/photos/reorder", { method: "POST", body: { photoIds } })
};

export const discoveryApi = {
  feed: (limit = 10) => apiRequest<{ profiles: DiscoverCard[] }>(`/discover${query({ limit })}`),
  swipe: (targetUserId: string, action: SwipeAction) =>
    apiRequest<SwipeResponse>("/swipes", { method: "POST", body: { targetUserId, action } }),
  undo: () => apiRequest<{ profile: PublicProfile }>("/swipes/undo", { method: "POST" }),
  unmatch: (matchId: string) => apiRequest<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: (before?: string | null) =>
    apiRequest<Page<"conversations", Conversation>>(`/conversations${query({ before, limit: 20 })}`),
  unreadCount: () => apiRequest<{ unreadCount: number }>("/conversations/unread-count"),
  conversation: (id: string) => apiRequest<Conversation>(`/conversations/${id}`),
  messages: (id: string, before?: string | null) =>
    apiRequest<MessagesPage>(`/conversations/${id}/messages${query({ before, limit: 30 })}`),
  send: (id: string, content: string, clientMessageId: string) =>
    apiRequest<Message>(`/conversations/${id}/messages`, { method: "POST", body: { content, clientMessageId } }),
  markRead: (id: string) => apiRequest<{ readAt: string }>(`/conversations/${id}/read`, { method: "POST" }),
  hide: (id: string) => apiRequest<void>(`/conversations/${id}`, { method: "DELETE" })
};

export const safetyApi = {
  blocks: () => apiRequest<{ blocks: BlockedUser[] }>("/blocks"),
  block: (userId: string) => apiRequest<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    apiRequest<void>("/reports", { method: "POST", body: { userId, reason, details } })
};

export const notificationsApi = {
  list: (before?: string | null) => apiRequest<NotificationsPage>(`/notifications${query({ before, limit: 20 })}`),
  unreadCount: () => apiRequest<{ unreadCount: number }>("/notifications/unread-count"),
  markRead: (id: string) => apiRequest<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest<void>("/notifications/read-all", { method: "POST" })
};
