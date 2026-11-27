import { apiRequest } from "./client";
import type {
  AppNotification,
  AuthSession,
  AuthUser,
  BlockedUser,
  ChatMessage,
  Gender,
  Interest,
  MatchSummary,
  MatchesPage,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Photo,
  ProfileCard,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

const q = (params: Record<string, string | number | undefined>) => {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== "")
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  return parts.length ? `?${parts.join("&")}` : "";
};

// --- Auth -------------------------------------------------------------------

export const authApi = {
  register: (email: string, password: string) =>
    apiRequest<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    apiRequest<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    apiRequest<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  me: () => apiRequest<AuthUser>("/me"),
  forgotPassword: (email: string) =>
    apiRequest<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, password: string) =>
    apiRequest<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false }),
  deleteAccount: (password: string) => apiRequest<void>("/me", { method: "DELETE", body: { password } })
};

// --- Profile ----------------------------------------------------------------

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  city: string;
};

export const profileApi = {
  mine: () => apiRequest<MyProfile>("/me/profile"),
  save: (input: ProfileInput) => apiRequest<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (input: { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number }) =>
    apiRequest<MyProfile>("/me/preferences", { method: "PUT", body: input }),
  saveInterests: (interestIds: number[]) =>
    apiRequest<MyProfile>("/me/interests", { method: "PUT", body: { interestIds } }),
  setLocation: (latitude: number, longitude: number) =>
    apiRequest<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => apiRequest<MyProfile>("/me/location", { method: "DELETE" }),
  setPrivacy: (isVisible: boolean, showDistance: boolean) =>
    apiRequest<MyProfile>("/me/privacy", { method: "PUT", body: { isVisible, showDistance } }),
  interests: async () => (await apiRequest<{ interests: Interest[] }>("/interests")).interests,
  publicProfile: (userId: string) => apiRequest<ProfileCard>(`/profiles/${userId}`)
};

// --- Photos -----------------------------------------------------------------

export type UploadablePhoto = { uri: string; mimeType: string; fileName: string; webFile?: Blob };

function photoForm(photo: UploadablePhoto): FormData {
  const form = new FormData();
  if (photo.webFile) {
    form.append("photo", photo.webFile, photo.fileName);
  } else {
    // React Native's FormData accepts this file descriptor shape.
    form.append("photo", { uri: photo.uri, name: photo.fileName, type: photo.mimeType } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  upload: async (photo: UploadablePhoto) =>
    (await apiRequest<{ photos: Photo[] }>("/me/photos", { method: "POST", body: photoForm(photo), timeoutMs: 60_000 })).photos,
  replace: async (photoId: string, photo: UploadablePhoto) =>
    (await apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "PUT", body: photoForm(photo), timeoutMs: 60_000 }))
      .photos,
  remove: async (photoId: string) =>
    (await apiRequest<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" })).photos,
  reorder: async (photoIds: string[]) =>
    (await apiRequest<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })).photos
};

// --- Discovery & matches ----------------------------------------------------

export const discoveryApi = {
  feed: async (limit = 10) => (await apiRequest<{ profiles: ProfileCard[] }>(`/discovery${q({ limit })}`)).profiles,
  swipe: (targetUserId: string, action: SwipeAction) =>
    apiRequest<SwipeResult>("/discovery/swipes", { method: "POST", body: { targetUserId, action } })
};

export const matchesApi = {
  list: (cursor?: string) => apiRequest<MatchesPage>(`/matches${q({ cursor, limit: 20 })}`),
  get: (matchId: string) => apiRequest<MatchSummary>(`/matches/${matchId}`),
  unmatch: (matchId: string) => apiRequest<void>(`/matches/${matchId}`, { method: "DELETE" }),
  messages: (matchId: string, cursor?: string) =>
    apiRequest<MessagesPage>(`/matches/${matchId}/messages${q({ cursor, limit: 30 })}`),
  send: (matchId: string, content: string, clientId: string) =>
    apiRequest<ChatMessage>(`/matches/${matchId}/messages`, { method: "POST", body: { content, clientId } }),
  markRead: (matchId: string) => apiRequest<{ readAt: string }>(`/matches/${matchId}/read`, { method: "POST" })
};

// --- Safety -----------------------------------------------------------------

export const safetyApi = {
  block: (userId: string) => apiRequest<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: async () => (await apiRequest<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    apiRequest<{ status: string; blocked: boolean }>("/reports", { method: "POST", body: { userId, reason, details, block } })
};

// --- Notifications ----------------------------------------------------------

export const notificationsApi = {
  list: (cursor?: string) => apiRequest<NotificationsPage>(`/notifications${q({ cursor, limit: 20 })}`),
  unreadCount: async () => (await apiRequest<{ unread: number }>("/notifications/unread-count")).unread,
  markRead: (id: string) => apiRequest<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest<void>("/notifications/read-all", { method: "POST" })
};
