import { Platform } from "react-native";
import { request } from "./client";
import type {
  Account,
  AuthResponse,
  BlockedUser,
  ConversationsPage,
  Gender,
  Interest,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Photo,
  Preferences,
  ProfileCard,
  RelationshipGoal,
  ReportReason,
  SwipeDecision,
  SwipeResponse
} from "./types";

// --- Auth -------------------------------------------------------------------

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) =>
    request<void>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),
  resetPassword: (email: string, code: string, password: string) =>
    request<void>("/auth/password/reset", { method: "POST", body: { email, code, password }, auth: false }),
  me: () => request<Account>("/me"),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } })
};

// --- Profile ----------------------------------------------------------------

export type ProfileInput = {
  firstName: string;
  birthdate: string;
  gender: Gender;
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  relationshipGoal: RelationshipGoal;
  city: string;
};

export const profileApi = {
  mine: () => request<MyProfile>("/me/profile"),
  save: (input: ProfileInput) => request<MyProfile>("/me/profile", { method: "PUT", body: input }),
  setLocation: (latitude: number, longitude: number, city?: string) =>
    request<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  setPrivacy: (isDiscoverable: boolean, showDistance: boolean) =>
    request<MyProfile>("/me/privacy", { method: "PUT", body: { isDiscoverable, showDistance } }),
  setPreferences: (prefs: Preferences) => request<Preferences>("/me/preferences", { method: "PUT", body: prefs }),
  interests: () => request<{ interests: Interest[] }>("/interests"),
  setInterests: (interestIds: number[]) =>
    request<{ interests: Interest[] }>("/me/interests", { method: "PUT", body: { interestIds } }),
  card: (userId: string) => request<ProfileCard>(`/profiles/${userId}`)
};

// --- Photos -----------------------------------------------------------------

export type LocalImage = { uri: string; mimeType?: string; file?: File | Blob };

async function photoForm(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = image.file ?? (await (await fetch(image.uri)).blob());
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors.
    form.append("photo", { uri: image.uri, name: "photo.jpg", type: image.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  list: () => request<{ photos: Photo[] }>("/me/photos"),
  upload: async (image: LocalImage) =>
    request<Photo>("/me/photos", { method: "POST", form: await photoForm(image), timeoutMs: 60_000 }),
  replace: async (photoId: string, image: LocalImage) =>
    request<Photo>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(image), timeoutMs: 60_000 }),
  remove: (photoId: string) => request<void>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => request<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};

// --- Matching ---------------------------------------------------------------

export const matchingApi = {
  discover: (limit = 10) => request<{ profiles: ProfileCard[] }>("/discovery", { query: { limit } }),
  swipe: (targetUserId: string, decision: SwipeDecision) =>
    request<SwipeResponse>("/swipes", { method: "POST", body: { targetUserId, decision } }),
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" })
};

// --- Chat -------------------------------------------------------------------

export const chatApi = {
  conversations: (before?: string) => request<ConversationsPage>("/conversations", { query: { before, limit: 30 } }),
  messages: (conversationId: string, before?: string) =>
    request<MessagesPage>(`/conversations/${conversationId}/messages`, { query: { before, limit: 30 } }),
  send: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => request<void>(`/conversations/${conversationId}/read`, { method: "POST" }),
  hide: (conversationId: string) => request<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};

// --- Notifications ----------------------------------------------------------

export const notificationsApi = {
  list: (before?: string) => request<NotificationsPage>("/notifications", { query: { before, limit: 30 } }),
  unreadCount: () => request<{ unreadCount: number }>("/notifications/unread-count"),
  markRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request<void>("/notifications/read-all", { method: "POST" })
};

// --- Safety -----------------------------------------------------------------

export const safetyApi = {
  blocks: () => request<{ blocks: BlockedUser[] }>("/blocks"),
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    request<{ id: string; blocked: boolean }>("/reports", { method: "POST", body: { userId, reason, details } })
};

// --- Realtime ---------------------------------------------------------------

export const realtimeApi = {
  ticket: () => request<{ ticket: string; expiresIn: number }>("/realtime/ticket", { method: "POST" })
};
