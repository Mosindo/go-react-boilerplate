import { apiRequest } from "./client";
import type {
  BlockedUser,
  Card,
  Conversation,
  Interest,
  MatchItem,
  Message,
  MessagesPage,
  NotificationsPage,
  OwnProfile,
  PhotoRef,
  Preferences,
  ProfileInput,
  ReportReason,
  SwipeResult
} from "./types";

// Profile & preferences
export const getOwnProfile = () => apiRequest<OwnProfile>("/me/profile");
export const saveProfile = (input: ProfileInput) => apiRequest<OwnProfile>("/me/profile", { method: "PUT", body: input });
export const savePreferences = (prefs: Preferences) => apiRequest<OwnProfile>("/me/preferences", { method: "PUT", body: prefs });
export const listInterests = () => apiRequest<{ interests: Interest[] }>("/interests").then((r) => r.interests);
export const getProfile = (id: string) => apiRequest<Card>(`/profiles/${id}`);

// Photos
type PhotosResponse = { photos: PhotoRef[] };

function photoForm(uri: string, mimeType?: string): FormData {
  const form = new FormData();
  // React Native's FormData accepts a { uri, name, type } descriptor for file parts.
  form.append("photo", { uri, name: "photo.jpg", type: mimeType ?? "image/jpeg" } as unknown as Blob);
  return form;
}

export const uploadPhoto = (uri: string, mimeType?: string) =>
  apiRequest<PhotosResponse>("/me/photos", { method: "POST", body: photoForm(uri, mimeType), timeoutMs: 60_000 }).then((r) => r.photos);
export const replacePhoto = (id: string, uri: string, mimeType?: string) =>
  apiRequest<PhotosResponse>(`/me/photos/${id}`, { method: "PUT", body: photoForm(uri, mimeType), timeoutMs: 60_000 }).then(
    (r) => r.photos
  );
export const deletePhoto = (id: string) => apiRequest<PhotosResponse>(`/me/photos/${id}`, { method: "DELETE" }).then((r) => r.photos);
export const reorderPhotos = (ids: string[]) =>
  apiRequest<PhotosResponse>("/me/photos/order", { method: "PUT", body: { ids } }).then((r) => r.photos);

// Discovery & matching
export const getFeed = (limit = 10) => apiRequest<{ profiles: Card[] }>(`/discover?limit=${limit}`).then((r) => r.profiles);
export const getReceivedLikes = () => apiRequest<{ profiles: Card[] }>("/likes/received?limit=20").then((r) => r.profiles);
export const swipe = (id: string, kind: "like" | "pass") =>
  apiRequest<SwipeResult>(`/discover/${id}/${kind}`, { method: "POST", silent: true });
export const getMatches = () => apiRequest<{ matches: MatchItem[] }>("/matches?limit=20").then((r) => r.matches);
export const unmatch = (matchId: string) => apiRequest(`/matches/${matchId}`, { method: "DELETE" });

// Chat
export const getConversations = () => apiRequest<{ conversations: Conversation[] }>("/conversations").then((r) => r.conversations);
export const getMessages = (conversationId: string, before?: string) =>
  apiRequest<MessagesPage>(`/conversations/${conversationId}/messages?limit=30${before ? `&before=${encodeURIComponent(before)}` : ""}`, {
    silent: true
  });
export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body }, silent: true });
export const markConversationRead = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}/read`, { method: "POST", silent: true });
export const hideConversation = (conversationId: string) => apiRequest(`/conversations/${conversationId}`, { method: "DELETE" });

// Notifications
export const getNotifications = (before?: string) =>
  apiRequest<NotificationsPage>(`/notifications?limit=20${before ? `&before=${encodeURIComponent(before)}` : ""}`);
export const markNotificationRead = (id: string) => apiRequest(`/notifications/${id}/read`, { method: "POST", silent: true });
export const markAllNotificationsRead = () => apiRequest("/notifications/read-all", { method: "POST", silent: true });

// Safety
export const getBlocked = () => apiRequest<{ blocked: BlockedUser[] }>("/blocks").then((r) => r.blocked);
export const blockUser = (userId: string) => apiRequest("/blocks", { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" });
export const reportUser = (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
  apiRequest("/reports", { method: "POST", body: { userId, reason, details, alsoBlock } });
