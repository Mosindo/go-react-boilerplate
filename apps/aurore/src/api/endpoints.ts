import { request } from "./client";
import type {
  Account,
  AppNotification,
  AuthSession,
  BlockedUser,
  Card,
  Conversation,
  Gender,
  Interest,
  Match,
  Message,
  PhotoRef,
  Preferences,
  ReportReason,
  SelfProfile,
  SwipeAction,
  SwipeResult
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthSession>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthSession>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgot: (email: string) => request<void>("/auth/forgot", { method: "POST", body: { email }, auth: false }),
  reset: (email: string, code: string, newPassword: string) =>
    request<void>("/auth/reset", { method: "POST", body: { email, code, newPassword }, auth: false }),
  me: () => request<Account>("/me"),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } })
};

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio?: string;
  city?: string;
  isVisible?: boolean;
  showDistance?: boolean;
};

export const profileApi = {
  get: () => request<SelfProfile>("/me/profile"),
  save: (input: ProfileInput) => request<SelfProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (p: Preferences) => request<SelfProfile>("/me/preferences", { method: "PUT", body: p }),
  saveLocation: (latitude: number, longitude: number, city?: string) =>
    request<SelfProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city: city ?? "" } }),
  saveInterests: (interestIds: number[]) =>
    request<SelfProfile>("/me/interests", { method: "PUT", body: { interestIds } }),
  interests: async () => (await request<{ interests: Interest[] }>("/interests")).interests
};

export type UploadFile = { uri: string; name: string; type: string; blob?: Blob };

function toForm(file: UploadFile): FormData {
  const form = new FormData();
  if (file.blob) {
    form.append("file", file.blob, file.name);
  } else {
    // React Native's FormData accepts this {uri,name,type} shape for local files.
    form.append("file", { uri: file.uri, name: file.name, type: file.type } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  list: async () => (await request<{ photos: PhotoRef[] }>("/me/photos")).photos,
  add: (file: UploadFile) => request<PhotoRef>("/me/photos", { method: "POST", form: toForm(file) }),
  replace: (id: string, file: UploadFile) =>
    request<PhotoRef>(`/me/photos/${id}`, { method: "PUT", form: toForm(file) }),
  remove: (id: string) => request<void>(`/me/photos/${id}`, { method: "DELETE" }),
  reorder: async (photoIds: string[]) =>
    (await request<{ photos: PhotoRef[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })).photos
};

export const discoverApi = {
  feed: async (limit = 10) => (await request<{ profiles: Card[] }>(`/discover?limit=${limit}`)).profiles,
  swipe: (userId: string, action: SwipeAction) =>
    request<SwipeResult>("/discover/swipes", { method: "POST", body: { userId, action } }),
  undo: (userId: string) => request<void>(`/discover/swipes/${userId}`, { method: "DELETE" }),
  profile: (userId: string) => request<Card>(`/profiles/${userId}`),
  matches: async () => (await request<{ matches: Match[] }>("/matches?limit=100")).matches,
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: async () => (await request<{ conversations: Conversation[] }>("/conversations?limit=100")).conversations,
  messages: async (conversationId: string, before?: string) => {
    const q = before ? `&before=${encodeURIComponent(before)}` : "";
    return (await request<{ messages: Message[] }>(`/conversations/${conversationId}/messages?limit=30${q}`)).messages;
  },
  send: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) =>
    request<{ read: number }>(`/conversations/${conversationId}/read`, { method: "POST" }),
  hide: (conversationId: string) => request<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const notificationsApi = {
  list: () =>
    request<{ notifications: AppNotification[]; unreadCount: number }>("/notifications?limit=50"),
  unreadCount: async () => (await request<{ unreadCount: number }>("/notifications/unread-count")).unreadCount,
  markRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request<void>("/notifications/read-all", { method: "POST" })
};

export const moderationApi = {
  blocked: async () => (await request<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details, alsoBlock } })
};

export const realtimeApi = {
  ticket: async () => (await request<{ ticket: string }>("/realtime/ticket", { method: "POST" })).ticket
};
