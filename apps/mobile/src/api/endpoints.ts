import { api } from "./client";
import type {
  AppNotification,
  AuthResponse,
  BlockedUser,
  Candidate,
  Conversation,
  Interest,
  Message,
  Overview,
  PhotoRef,
  Preferences,
  Privacy,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

export const authApi = {
  register: (email: string, password: string) =>
    api<AuthResponse>("/auth/register", { body: { email, password }, anonymous: true }),
  login: (email: string, password: string) =>
    api<AuthResponse>("/auth/login", { body: { email, password }, anonymous: true }),
  logout: (refreshToken: string) =>
    api("/auth/logout", { body: { refreshToken }, anonymous: true }),
  forgotPassword: (email: string) =>
    api("/auth/password/forgot", { body: { email }, anonymous: true }),
  resetPassword: (token: string, newPassword: string) =>
    api("/auth/password/reset", { body: { token, newPassword }, anonymous: true }),
  changePassword: (currentPassword: string, newPassword: string) =>
    api("/me/password", { body: { currentPassword, newPassword } }),
  deleteAccount: (password: string) => api("/me/delete", { body: { password } })
};

export const profileApi = {
  overview: () => api<Overview>("/me/profile"),
  interests: () => api<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  saveProfile: (input: ProfileInput) =>
    api<Overview>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (p: Preferences) => api<Overview>("/me/preferences", { method: "PUT", body: p }),
  savePrivacy: (p: Privacy) => api<Overview>("/me/privacy", { method: "PUT", body: p }),
  saveLocation: (latitude: number, longitude: number) =>
    api<Overview>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  clearLocation: () => api<Overview>("/me/location", { method: "DELETE" })
};

export type UploadFile = { uri: string; name: string; type: string };

function photoForm(file: UploadFile): FormData {
  const form = new FormData();
  // React Native's FormData accepts {uri,name,type}; the DOM typing does not know it.
  form.append("photo", file as unknown as Blob);
  return form;
}

async function webAwareForm(file: UploadFile): Promise<FormData> {
  // On web the picker returns a blob: URL that must be turned into a real Blob.
  if (file.uri.startsWith("blob:") || file.uri.startsWith("data:")) {
    const blob = await (await fetch(file.uri)).blob();
    const form = new FormData();
    form.append("photo", blob, file.name);
    return form;
  }
  return photoForm(file);
}

export const photosApi = {
  upload: async (file: UploadFile) =>
    api<PhotoRef>("/me/photos", { form: await webAwareForm(file) }),
  replace: async (id: string, file: UploadFile) =>
    api<PhotoRef>(`/me/photos/${id}`, { method: "PUT", form: await webAwareForm(file) }),
  remove: (id: string) => api(`/me/photos/${id}`, { method: "DELETE" }),
  reorder: (ids: string[]) =>
    api<{ photos: PhotoRef[] }>("/me/photos/order", { method: "PUT", body: { ids } })
};

export const discoveryApi = {
  feed: (limit = 10) =>
    api<{ profiles: Candidate[] }>(`/discover?limit=${limit}`).then((r) => r.profiles),
  profile: (id: string) => api<Candidate>(`/profiles/${id}`),
  swipe: (userId: string, action: SwipeAction) =>
    api<SwipeResult>("/swipes", { body: { userId, action } }),
  unmatch: (matchId: string) => api(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  conversations: () =>
    api<{ conversations: Conversation[]; nextCursor?: string }>("/conversations?limit=50").then(
      (r) => r.conversations
    ),
  messages: (id: string, before?: number) =>
    api<{ messages: Message[]; hasMore: boolean }>(
      `/conversations/${id}/messages?limit=30${before ? `&before=${before}` : ""}`
    ),
  send: (id: string, body: string) =>
    api<Message>(`/conversations/${id}/messages`, { body: { body } }),
  markRead: (id: string) => api(`/conversations/${id}/read`, { method: "POST" }),
  hide: (id: string) => api(`/conversations/${id}`, { method: "DELETE" })
};

export const safetyApi = {
  block: (userId: string) => api("/blocks", { body: { userId } }),
  unblock: (userId: string) => api(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
    api("/reports", { body: { userId, reason, details, alsoBlock } })
};

export const notificationsApi = {
  list: (cursor?: string) =>
    api<{ notifications: AppNotification[]; nextCursor?: string }>(
      `/notifications?limit=20${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`
    ),
  unread: () => api<{ unread: number }>("/notifications/unread-count").then((r) => r.unread),
  markRead: (id: string) => api(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: () => api<{ ticket: string }>("/ws/ticket", { method: "POST" }).then((r) => r.ticket)
};
