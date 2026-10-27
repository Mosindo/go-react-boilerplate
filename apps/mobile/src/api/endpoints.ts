import { request } from "./client";
import type {
  AuthSession,
  AuthUser,
  BlockedUser,
  ConversationPage,
  Interest,
  MatchPage,
  Message,
  MessagePage,
  MyProfile,
  NotificationPage,
  Photo,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

function query(params: Record<string, string | number | undefined>): string {
  const parts = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== "")
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`);
  return parts.length ? `?${parts.join("&")}` : "";
}

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthSession>("/auth/register", {
      method: "POST",
      body: { email, password },
      anonymous: true
    }),
  login: (email: string, password: string) =>
    request<AuthSession>("/auth/login", {
      method: "POST",
      body: { email, password },
      anonymous: true
    }),
  logout: (refreshToken: string) =>
    request("/auth/logout", { method: "POST", body: { refreshToken }, anonymous: true }),
  me: () => request<AuthUser>("/me"),
  forgotPassword: (email: string) =>
    request("/auth/forgot-password", { method: "POST", body: { email }, anonymous: true }),
  resetPassword: (email: string, code: string, newPassword: string) =>
    request("/auth/reset-password", {
      method: "POST",
      body: { email, code, newPassword },
      anonymous: true
    }),
  changePassword: (currentPassword: string, newPassword: string) =>
    request("/auth/change-password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => request("/me", { method: "DELETE", body: { password } })
};

export const profileApi = {
  get: () => request<MyProfile>("/profile/me"),
  save: (input: ProfileInput) => request<MyProfile>("/profile/me", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) =>
    request<MyProfile>("/profile/preferences", { method: "PUT", body: prefs }),
  savePrivacy: (privacy: { discoverable?: boolean; showDistance?: boolean }) =>
    request<MyProfile>("/profile/privacy", { method: "PUT", body: privacy }),
  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  addPhoto: (form: FormData) => request<Photo>("/photos", { method: "POST", form }),
  replacePhoto: (id: string, form: FormData) =>
    request<Photo>(`/photos/${id}`, { method: "PUT", form }),
  deletePhoto: (id: string) => request(`/photos/${id}`, { method: "DELETE" }),
  reorderPhotos: (ids: string[]) =>
    request<MyProfile>("/photos/order", { method: "PUT", body: { ids } })
};

export const discoveryApi = {
  feed: (limit = 10) =>
    request<{ profiles: PublicProfile[] }>(`/discover${query({ limit })}`).then((r) => r.profiles),
  swipe: (targetId: string, action: SwipeAction) =>
    request<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } }),
  matches: (cursor?: string) => request<MatchPage>(`/matches${query({ limit: 30, cursor })}`),
  unmatch: (matchId: string) => request(`/matches/${matchId}`, { method: "DELETE" }),
  profile: (userId: string) => request<PublicProfile>(`/users/${userId}/profile`)
};

export const chatApi = {
  conversations: (cursor?: string) =>
    request<ConversationPage>(`/conversations${query({ limit: 30, cursor })}`),
  messages: (conversationId: string, cursor?: string) =>
    request<MessagePage>(
      `/conversations/${conversationId}/messages${query({ limit: 40, cursor })}`
    ),
  send: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, {
      method: "POST",
      body: { body }
    }),
  markRead: (conversationId: string) =>
    request<{ upToMessageId: number }>(`/conversations/${conversationId}/read`, { method: "POST" }),
  clear: (conversationId: string) =>
    request(`/conversations/${conversationId}`, { method: "DELETE" })
};

export const notificationsApi = {
  list: (cursor?: string) =>
    request<NotificationPage>(`/notifications${query({ limit: 30, cursor })}`),
  markRead: (id: string) => request(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request("/notifications/read", { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => request("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => request<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    request<{ id: string }>("/reports", {
      method: "POST",
      body: { userId, reason, details, block }
    })
};
