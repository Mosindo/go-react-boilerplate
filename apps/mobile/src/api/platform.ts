// Typed calls for every non-auth endpoint. UI code goes through these (or the
// React Query hooks built on them) and never calls fetch directly.
import { apiRequest, type FileUpload } from "./client";
import { endpoints } from "./endpoints";
import type {
  AppNotification,
  BlockedUser,
  Card,
  Conversation,
  Gender,
  Interest,
  Message,
  MessagesPage,
  MyProfileResponse,
  NotificationsResponse,
  PhotoView,
  Preferences,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResponse
} from "./types";

// ---- profile ----

export const getMyProfile = () => apiRequest<MyProfileResponse>(endpoints.profile.me);

export const saveProfile = (input: ProfileInput) => apiRequest<MyProfileResponse>(endpoints.profile.me, { method: "PUT", body: input });

export const savePreferences = (input: Preferences) =>
  apiRequest<MyProfileResponse>(endpoints.profile.preferences, { method: "PUT", body: input });

export const saveLocation = (latitude: number, longitude: number, city?: string) =>
  apiRequest<MyProfileResponse>(endpoints.profile.location, { method: "PUT", body: { latitude, longitude, city: city ?? "" } });

export const clearLocation = () => apiRequest<MyProfileResponse>(endpoints.profile.location, { method: "DELETE" });

export const savePrivacy = (input: { isVisible?: boolean; showDistance?: boolean }) =>
  apiRequest<MyProfileResponse>(endpoints.profile.privacy, { method: "PATCH", body: input });

export const listInterests = async (): Promise<Interest[]> =>
  (await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests)).interests;

export const getProfile = (userId: string) => apiRequest<Card>(endpoints.profile.of(userId));

// ---- photos ----

type PhotosResponse = { photos: PhotoView[] };

export const listPhotos = async () => (await apiRequest<PhotosResponse>(endpoints.photos.list)).photos;

export const uploadPhoto = async (file: FileUpload) =>
  (await apiRequest<PhotosResponse>(endpoints.photos.list, { method: "POST", file })).photos;

export const replacePhoto = async (id: string, file: FileUpload) =>
  (await apiRequest<PhotosResponse>(endpoints.photos.one(id), { method: "PUT", file })).photos;

export const deletePhoto = async (id: string) => (await apiRequest<PhotosResponse>(endpoints.photos.one(id), { method: "DELETE" })).photos;

export const setMainPhoto = async (id: string) => (await apiRequest<PhotosResponse>(endpoints.photos.main(id), { method: "POST" })).photos;

export const reorderPhotos = async (ids: string[]) =>
  (await apiRequest<PhotosResponse>(endpoints.photos.order, { method: "PUT", body: { ids } })).photos;

// ---- discovery & matching ----

export const discover = async (exclude: string[], limit = 10): Promise<Card[]> =>
  (
    await apiRequest<{ profiles: Card[] }>(endpoints.matching.discover, {
      query: { limit, exclude: exclude.join(",") }
    })
  ).profiles;

export const swipe = (userId: string, action: SwipeAction) =>
  apiRequest<SwipeResponse>(endpoints.matching.swipes, { method: "POST", body: { userId, action } });

export const unmatch = (userId: string) => apiRequest(endpoints.matching.unmatch(userId), { method: "DELETE" });

// ---- chat ----

export const listConversations = async (): Promise<Conversation[]> =>
  (await apiRequest<{ conversations: Conversation[] }>(endpoints.chats.list, { query: { limit: 50 } })).conversations;

export const listMessages = (conversationId: string, before?: string) =>
  apiRequest<MessagesPage>(endpoints.chats.messages(conversationId), { query: { limit: 30, before } });

export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(endpoints.chats.messages(conversationId), { method: "POST", body: { body } });

export const markConversationRead = (conversationId: string) =>
  apiRequest<{ readAt: string }>(endpoints.chats.read(conversationId), { method: "POST" });

export const hideConversation = (conversationId: string) => apiRequest(endpoints.chats.one(conversationId), { method: "DELETE" });

// ---- notifications ----

export const listNotifications = () => apiRequest<NotificationsResponse>(endpoints.notifications.list, { query: { limit: 50 } });

export const markNotificationRead = (id: string) => apiRequest<AppNotification>(endpoints.notifications.read(id), { method: "POST" });

export const markAllNotificationsRead = () => apiRequest(endpoints.notifications.readAll, { method: "POST" });

// ---- safety ----

export const listBlocked = async (): Promise<BlockedUser[]> =>
  (await apiRequest<{ blocked: BlockedUser[] }>(endpoints.safety.blocks)).blocked;

export const blockUser = (userId: string) => apiRequest(endpoints.safety.block(userId), { method: "POST" });

export const unblockUser = (userId: string) => apiRequest(endpoints.safety.block(userId), { method: "DELETE" });

export const reportUser = (userId: string, reason: ReportReason, details: string) =>
  apiRequest(endpoints.safety.reports, { method: "POST", body: { userId, reason, details } });

// ---- realtime ----

export const requestSocketTicket = async () => (await apiRequest<{ ticket: string }>(endpoints.realtime.ticket, { method: "POST" })).ticket;

export type { Gender };
