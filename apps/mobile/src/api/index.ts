import { request } from "./client";
import type {
  AppNotification,
  AuthResponse,
  BlockedUser,
  Card,
  Conversation,
  Interest,
  LikeReceived,
  Message,
  MessagesPage,
  MyProfile,
  PhotoRef,
  Preferences,
  Privacy,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult,
  User
} from "./types";

export const api = {
  register: (email: string, password: string) =>
    request<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) => request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => request<void>("/auth/forgot-password", { method: "POST", body: { email }, auth: false }),
  resetPassword: (token: string, password: string) =>
    request<void>("/auth/reset-password", { method: "POST", body: { token, password }, auth: false }),
  me: () => request<User>("/me"),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } }),

  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  myProfile: () => request<MyProfile>("/me/profile"),
  saveProfile: (input: ProfileInput) => request<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => request<void>("/me/preferences", { method: "PUT", body: prefs }),
  savePrivacy: (privacy: Partial<Privacy>) => request<void>("/me/privacy", { method: "PUT", body: privacy }),
  setLocation: (latitude: number, longitude: number) => request<void>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => request<void>("/me/location", { method: "DELETE" }),
  profileCard: (id: string) => request<Card>(`/profiles/${id}`),

  uploadPhoto: (form: FormData) => request<PhotoRef>("/me/photos", { method: "POST", form }),
  replacePhoto: (id: string, form: FormData) => request<PhotoRef>(`/me/photos/${id}/replace`, { method: "POST", form }),
  deletePhoto: (id: string) => request<void>(`/me/photos/${id}`, { method: "DELETE" }),
  reorderPhotos: (ids: string[]) => request<void>("/me/photos/order", { method: "PUT", body: { ids } }),

  discover: (limit = 10) => request<{ profiles: Card[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetId: string, action: SwipeAction) => request<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } }),
  likesReceived: () => request<{ likes: LikeReceived[] }>("/likes/received?limit=50").then((r) => r.likes),
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" }),

  conversations: () => request<{ conversations: Conversation[] }>("/conversations?limit=50").then((r) => r.conversations),
  messages: (conversationId: string, before?: number) =>
    request<MessagesPage>(`/conversations/${conversationId}/messages?limit=40${before ? `&before=${before}` : ""}`),
  sendMessage: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => request<void>(`/conversations/${conversationId}/read`, { method: "POST" }),
  hideConversation: (conversationId: string) => request<void>(`/conversations/${conversationId}`, { method: "DELETE" }),

  notifications: () => request<{ notifications: AppNotification[]; unreadCount: number }>("/notifications?limit=50"),
  markNotificationRead: (id: string) => request<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllNotificationsRead: () => request<void>("/notifications/read", { method: "POST" }),

  blocks: () => request<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details } })
};
