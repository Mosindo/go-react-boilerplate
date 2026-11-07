import { Platform } from "react-native";
import { request } from "./http";
import type {
  AuthResponse,
  BlockedUser,
  Card,
  ConversationSummary,
  Interest,
  MatchView,
  Message,
  MessagesPage,
  MyProfile,
  NotificationItem,
  PhotoView,
  ProfileInput,
  ReportReason,
  SwipeResponse,
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthResponse>("/auth/register", { method: "POST", body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    request<void>("/auth/logout", { method: "POST", body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) =>
    request<void>("/auth/forgot-password", { method: "POST", body: { email }, auth: false }),
  resetPassword: (token: string, newPassword: string) =>
    request<void>("/auth/reset-password", { method: "POST", body: { token, newPassword }, auth: false }),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<void>("/me/password", { method: "POST", body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => request<void>("/me", { method: "DELETE", body: { password } }),
};

export const profileApi = {
  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  mine: () => request<MyProfile>("/profile/me"),
  save: (input: ProfileInput) => request<MyProfile>("/profile/me", { method: "PUT", body: input }),
  view: (userId: string) => request<Card>(`/profiles/${userId}`),
  deletePhoto: (id: string) => request<void>(`/profile/photos/${id}`, { method: "DELETE" }),
  reorderPhotos: (ids: string[]) => request<void>("/profile/photos/order", { method: "PUT", body: { ids } }),
  uploadPhoto: async (uri: string, mimeType?: string | null): Promise<PhotoView> => {
    const form = new FormData();
    const type = mimeType ?? "image/jpeg";
    if (Platform.OS === "web") {
      const blob = await (await fetch(uri)).blob();
      form.append("photo", blob, "photo.jpg");
    } else {
      // React Native's FormData accepts a {uri,name,type} descriptor in place of a Blob.
      form.append("photo", { uri, name: "photo.jpg", type } as unknown as Blob);
    }
    return request<PhotoView>("/profile/photos", { method: "POST", form });
  },
};

export const discoveryApi = {
  feed: (limit = 10) => request<{ profiles: Card[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  swipe: (userId: string, action: "like" | "pass") =>
    request<SwipeResponse>("/swipes", { method: "POST", body: { userId, action } }),
  likesReceived: () => request<{ profiles: Card[] }>("/likes/received?limit=50").then((r) => r.profiles),
  matches: () => request<{ matches: MatchView[] }>("/matches?limit=100").then((r) => r.matches),
  unmatch: (matchId: string) => request<void>(`/matches/${matchId}`, { method: "DELETE" }),
};

export const chatApi = {
  conversations: () =>
    request<{ conversations: ConversationSummary[] }>("/conversations?limit=100").then(
      (r) => r.conversations,
    ),
  messages: (conversationId: string, before?: number) =>
    request<MessagesPage>(
      `/conversations/${conversationId}/messages?limit=30${before ? `&before=${before}` : ""}`,
    ),
  send: (conversationId: string, body: string) =>
    request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) =>
    request<void>(`/conversations/${conversationId}/read`, { method: "POST" }),
  clear: (conversationId: string) => request<void>(`/conversations/${conversationId}`, { method: "DELETE" }),
};

export const notificationsApi = {
  list: () => request<{ notifications: NotificationItem[]; unreadCount: number }>("/notifications?limit=50"),
  readAll: () => request<void>("/notifications/read", { method: "POST" }),
};

export const safetyApi = {
  blocks: () => request<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, block: boolean) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details, block } }),
};
