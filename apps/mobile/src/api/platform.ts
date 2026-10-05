import { api } from "./client";
import { endpoints } from "./endpoints";
import type {
  AppNotification,
  BlockedUser,
  ConversationPreview,
  Interest,
  MatchPreview,
  Message,
  MessagesPage,
  MyProfile,
  NotificationsPage,
  Photo,
  Preferences,
  ProfileCard,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";
import { buildPhotoForm, type PickedImage } from "../lib/upload";

/** Typed facade over the REST API. UI code goes through these functions (or the hooks built on them). */

// ── Profile ──────────────────────────────────────────────────────────────────
export const getMyProfile = (signal?: AbortSignal) => api.request<MyProfile>(endpoints.myProfile, { signal });

export const saveProfile = (input: ProfileInput) => api.request<MyProfile>(endpoints.myProfile, { method: "PUT", body: input });

export const savePreferences = (preferences: Preferences) =>
  api.request<MyProfile>(endpoints.preferences, { method: "PUT", body: preferences });

export const saveLocation = (latitude: number, longitude: number, city: string) =>
  api.request<MyProfile>(endpoints.location, { method: "PUT", body: { latitude, longitude, city } });

export const savePrivacy = (isDiscoverable: boolean, showDistance: boolean) =>
  api.request<MyProfile>(endpoints.privacy, { method: "PUT", body: { isDiscoverable, showDistance } });

export const getInterests = async (): Promise<Interest[]> =>
  (await api.request<{ interests: Interest[] }>(endpoints.interests)).interests;

export const getProfile = (userId: string) => api.request<ProfileCard>(endpoints.profile(userId));

// ── Photos ───────────────────────────────────────────────────────────────────
export const uploadPhoto = async (image: PickedImage) =>
  api.request<Photo>(endpoints.photos, { method: "POST", form: await buildPhotoForm(image) });

export const replacePhoto = async (photoId: string, image: PickedImage) =>
  api.request<Photo>(endpoints.photo(photoId), { method: "PUT", form: await buildPhotoForm(image) });

export const deletePhoto = (photoId: string) => api.request<void>(endpoints.photo(photoId), { method: "DELETE" });

export const reorderPhotos = async (photoIds: string[]): Promise<Photo[]> =>
  (await api.request<{ photos: Photo[] }>(endpoints.photosOrder, { method: "PUT", body: { photoIds } })).photos;

// ── Discovery & matches ──────────────────────────────────────────────────────
export const getDiscover = async (limit: number, signal?: AbortSignal): Promise<ProfileCard[]> =>
  (await api.request<{ profiles: ProfileCard[] }>(`${endpoints.discover}?limit=${limit}`, { signal })).profiles;

export const swipe = (targetUserId: string, action: SwipeAction) =>
  api.request<SwipeResult>(endpoints.swipes, { method: "POST", body: { targetUserId, action } });

export const getMatches = async (signal?: AbortSignal): Promise<MatchPreview[]> =>
  (await api.request<{ matches: MatchPreview[] }>(`${endpoints.matches}?limit=100`, { signal })).matches;

export const unmatch = (matchId: string) => api.request<void>(endpoints.match(matchId), { method: "DELETE" });

// ── Chat ─────────────────────────────────────────────────────────────────────
export const getConversations = async (signal?: AbortSignal): Promise<ConversationPreview[]> =>
  (await api.request<{ conversations: ConversationPreview[] }>(`${endpoints.conversations}?limit=100`, { signal })).conversations;

export const getMessages = (conversationId: string, before?: string, signal?: AbortSignal) =>
  api.request<MessagesPage>(
    `${endpoints.messages(conversationId)}?limit=40${before ? `&before=${encodeURIComponent(before)}` : ""}`,
    { signal }
  );

export const sendMessage = (conversationId: string, content: string) =>
  api.request<Message>(endpoints.messages(conversationId), { method: "POST", body: { content } });

export const markConversationRead = (conversationId: string) =>
  api.request<{ unreadCount: number }>(endpoints.markRead(conversationId), { method: "POST" });

export const clearConversation = (conversationId: string) =>
  api.request<void>(endpoints.conversation(conversationId), { method: "DELETE" });

// ── Notifications ────────────────────────────────────────────────────────────
export const getNotifications = (signal?: AbortSignal) =>
  api.request<NotificationsPage>(`${endpoints.notifications}?limit=50`, { signal });

export const markNotificationRead = (id: string) =>
  api.request<AppNotification>(endpoints.notificationRead(id), { method: "POST" });

export const markAllNotificationsRead = () => api.request<void>(endpoints.notificationsReadAll, { method: "POST" });

// ── Safety ───────────────────────────────────────────────────────────────────
export const getBlocks = async (): Promise<BlockedUser[]> =>
  (await api.request<{ blocks: BlockedUser[] }>(endpoints.blocks)).blocks;

export const blockUser = (userId: string) => api.request<void>(endpoints.blocks, { method: "POST", body: { userId } });

export const unblockUser = (userId: string) => api.request<void>(endpoints.block(userId), { method: "DELETE" });

export const reportUser = (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
  api.request<void>(endpoints.reports, { method: "POST", body: { userId, reason, details, alsoBlock } });

