import { api } from "./client";
import type {
  BlockedUser,
  ChatsResponse,
  DiscoveryResponse,
  Interest,
  Match,
  Message,
  MessagesPage,
  NotificationsResponse,
  OwnProfile,
  Photo,
  Preferences,
  ProfileCard,
  ProfilePatch,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

// Profile ---------------------------------------------------------------

export const getMyProfile = () => api<OwnProfile>("/me/profile");
export const updateProfile = (patch: ProfilePatch) => api<OwnProfile>("/me/profile", { method: "PATCH", body: patch });
export const updateLocation = (latitude: number, longitude: number, city: string) =>
  api<OwnProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } });
export const updatePreferences = (prefs: Preferences) => api<OwnProfile>("/me/preferences", { method: "PUT", body: prefs });
export const listInterests = async () => (await api<{ interests: Interest[] }>("/interests")).interests;
export const getProfile = (userId: string) => api<ProfileCard>(`/profiles/${userId}`);

// Photos ----------------------------------------------------------------

type PhotosResponse = { photos: Photo[] };

export const uploadPhoto = async (form: FormData) => (await api<PhotosResponse>("/me/photos", { method: "POST", form })).photos;
export const replacePhoto = async (photoId: string, form: FormData) =>
  (await api<PhotosResponse>(`/me/photos/${photoId}`, { method: "PUT", form })).photos;
export const deletePhoto = async (photoId: string) => (await api<PhotosResponse>(`/me/photos/${photoId}`, { method: "DELETE" })).photos;
export const reorderPhotos = async (photoIds: string[]) =>
  (await api<PhotosResponse>("/me/photos/order", { method: "PUT", body: { photoIds } })).photos;

// Discovery & matching ---------------------------------------------------

export const getDiscovery = (limit = 10) => api<DiscoveryResponse>(`/discovery?limit=${limit}`);
export const swipe = (targetUserId: string, action: SwipeAction) =>
  api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } });
export const resetPasses = () => api<{ restored: number }>("/discovery/passes", { method: "DELETE" });
export const getLikesReceived = async () => (await api<{ profiles: ProfileCard[] }>("/likes/received?limit=50")).profiles;
export const getMatches = async () => (await api<{ matches: Match[] }>("/matches?limit=50")).matches;
export const unmatch = (userId: string) => api<void>(`/matches/${userId}`, { method: "DELETE" });

// Chat --------------------------------------------------------------------

export const getChats = () => api<ChatsResponse>("/chats?limit=50");
export const getMessages = (userId: string, before?: string) =>
  api<MessagesPage>(`/chats/${userId}/messages?limit=30${before ? `&before=${before}` : ""}`);
export const sendMessage = (userId: string, content: string) =>
  api<Message>(`/chats/${userId}/messages`, { method: "POST", body: { content } });
export const markRead = (userId: string) => api<void>(`/chats/${userId}/read`, { method: "POST" });
export const clearConversation = (userId: string) => api<void>(`/chats/${userId}`, { method: "DELETE" });

// Safety ------------------------------------------------------------------

export const blockUser = (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" });
export const getBlocks = async () => (await api<{ blocks: BlockedUser[] }>("/blocks")).blocks;
export const reportUser = (userId: string, reason: ReportReason, details: string) =>
  api<{ id: string }>("/reports", { method: "POST", body: { userId, reason, details } });

// Notifications -----------------------------------------------------------

export const getNotifications = () => api<NotificationsResponse>("/notifications?limit=50");
export const markNotificationRead = (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" });
export const markAllNotificationsRead = () => api<void>("/notifications/read-all", { method: "POST" });
