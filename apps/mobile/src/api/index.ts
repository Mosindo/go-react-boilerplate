import { request } from "./client";
import type {
  AccountUser,
  AuthResponse,
  BlockedUser,
  Card,
  Conversation,
  Gender,
  Interest,
  MatchSummary,
  Message,
  MessagePage,
  NotificationPage,
  Paged,
  Photo,
  Profile,
  ProfileInput,
  PublicProfile,
  ReportReason,
  Settings,
  SwipeResult,
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    request<AuthResponse>("POST", "/auth/register", { body: { email, password }, auth: false }),
  login: (email: string, password: string) =>
    request<AuthResponse>("POST", "/auth/login", { body: { email, password }, auth: false }),
  logout: (refreshToken: string) => request<void>("POST", "/auth/logout", { body: { refreshToken }, auth: false }),
  forgotPassword: (email: string) => request<void>("POST", "/auth/password/forgot", { body: { email }, auth: false }),
  resetPassword: (token: string, password: string) =>
    request<void>("POST", "/auth/password/reset", { body: { token, password }, auth: false }),
  me: () => request<AccountUser>("GET", "/me"),
  deleteAccount: (password: string) => request<void>("DELETE", "/me", { body: { password } }),
};

export const profileApi = {
  interests: () => request<{ items: Interest[] }>("GET", "/interests").then((r) => r.items),
  getMine: () => request<Profile>("GET", "/me/profile"),
  saveMine: (input: ProfileInput) => request<Profile>("PUT", "/me/profile", { body: input }),
  getSettings: () => request<Settings>("GET", "/me/settings"),
  saveSettings: (settings: Settings) => request<Settings>("PUT", "/me/settings", { body: settings }),
  getPublic: (userId: string) => request<PublicProfile>("GET", `/profiles/${userId}`),
};

export type UploadFile = { uri: string; name: string; type: string };

export const photosApi = {
  list: () => request<{ items: Photo[] }>("GET", "/me/photos").then((r) => r.items),
  upload: (file: UploadFile, replaceId?: string) => {
    const form = new FormData();
    // React Native accepte un objet {uri,name,type} comme partie de formulaire.
    form.append("file", file as unknown as Blob);
    return request<Photo>("POST", "/me/photos", { form, query: { replace: replaceId } });
  },
  remove: (id: string) => request<void>("DELETE", `/me/photos/${id}`),
  reorder: (ids: string[]) =>
    request<{ items: Photo[] }>("PUT", "/me/photos/order", { body: { ids } }).then((r) => r.items),
  makePrimary: (id: string) => request<{ items: Photo[] }>("PUT", `/me/photos/${id}/primary`).then((r) => r.items),
};

export const discoverApi = {
  feed: (limit = 10) => request<{ items: Card[] }>("GET", "/discover", { query: { limit } }).then((r) => r.items),
  like: (userId: string) => request<SwipeResult>("POST", `/profiles/${userId}/like`),
  pass: (userId: string) => request<SwipeResult>("POST", `/profiles/${userId}/pass`),
  matches: (cursor?: string) => request<Paged<MatchSummary>>("GET", "/matches", { query: { cursor } }),
  unmatch: (matchId: string) => request<void>("DELETE", `/matches/${matchId}`),
};

export const chatApi = {
  conversations: (cursor?: string) => request<Paged<Conversation>>("GET", "/conversations", { query: { cursor } }),
  messages: (conversationId: string, cursor?: string) =>
    request<MessagePage>("GET", `/conversations/${conversationId}/messages`, { query: { cursor, limit: 30 } }),
  send: (conversationId: string, body: string) =>
    request<Message>("POST", `/conversations/${conversationId}/messages`, { body: { body } }),
  markRead: (conversationId: string) => request<void>("POST", `/conversations/${conversationId}/read`),
  clear: (conversationId: string) => request<void>("DELETE", `/conversations/${conversationId}`),
};

export const moderationApi = {
  block: (userId: string) => request<void>("POST", `/users/${userId}/block`),
  unblock: (userId: string) => request<void>("DELETE", `/users/${userId}/block`),
  blocks: () => request<{ items: BlockedUser[] }>("GET", "/blocks").then((r) => r.items),
  report: (userId: string, reason: ReportReason, details: string) =>
    request<void>("POST", `/users/${userId}/report`, { body: { reason, details } }),
};

export const notificationsApi = {
  list: (cursor?: string) => request<NotificationPage>("GET", "/notifications", { query: { cursor } }),
  markRead: (ids: string[] = []) => request<void>("POST", "/notifications/read", { body: { ids } }),
};

export const GENDER_LABELS: Record<Gender, string> = {
  man: "Homme",
  woman: "Femme",
  nonbinary: "Non-binaire",
};
