import { Platform } from "react-native";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  BlockedUser,
  ConversationsPage,
  Interest,
  Me,
  MatchesPage,
  Message,
  MessagesPage,
  NotificationsPage,
  Photo,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult,
  UnreadSummary
} from "./types";

function page(params: Record<string, string | number | undefined>): string {
  const query = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== "")
    .map(([key, value]) => `${key}=${encodeURIComponent(String(value))}`)
    .join("&");
  return query ? `?${query}` : "";
}

export const getMe = () => apiRequest<Me>(endpoints.me);
export const saveProfile = (input: ProfileInput) => apiRequest<Me>(endpoints.profile, { method: "PUT", body: input });
export const savePreferences = (input: Preferences) => apiRequest<Me>(endpoints.preferences, { method: "PUT", body: input });
export const saveLocation = (latitude: number, longitude: number, city: string) =>
  apiRequest<Me>(endpoints.location, { method: "PUT", body: { latitude, longitude, city } });
export const clearLocation = () => apiRequest<Me>(endpoints.location, { method: "DELETE" });
export const listInterests = () => apiRequest<{ interests: Interest[] }>(endpoints.interests).then((r) => r.interests);
export const getProfile = (id: string) => apiRequest<PublicProfile>(endpoints.profileOf(id));

export const getDiscover = (limit = 10) => apiRequest<{ profiles: PublicProfile[] }>(`${endpoints.discover}${page({ limit })}`).then((r) => r.profiles);
export const swipe = (userId: string, action: SwipeAction) => apiRequest<SwipeResult>(endpoints.swipes, { method: "POST", body: { userId, action } });
export const getMatches = (cursor?: string) => apiRequest<MatchesPage>(`${endpoints.matches}${page({ limit: 30, cursor })}`);
export const unmatch = (matchId: string) => apiRequest(endpoints.match(matchId), { method: "DELETE" });

export const getConversations = (cursor?: string) => apiRequest<ConversationsPage>(`${endpoints.conversations}${page({ limit: 30, cursor })}`);
export const getUnread = () => apiRequest<UnreadSummary>(endpoints.conversationsUnread);
export const getMessages = (conversationId: string, cursor?: string) =>
  apiRequest<MessagesPage>(`${endpoints.messages(conversationId)}${page({ limit: 40, cursor })}`);
export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(endpoints.messages(conversationId), { method: "POST", body: { body } });
export const markRead = (conversationId: string) => apiRequest(endpoints.read(conversationId), { method: "POST" });
export const hideConversation = (conversationId: string) => apiRequest(endpoints.conversation(conversationId), { method: "DELETE" });

export const getNotifications = (cursor?: string) => apiRequest<NotificationsPage>(`${endpoints.notifications}${page({ limit: 30, cursor })}`);
export const getNotificationsUnread = () => apiRequest<{ unread: number }>(endpoints.notificationsUnread).then((r) => r.unread);
export const markNotificationRead = (id: string) => apiRequest(endpoints.notificationRead(id), { method: "POST" });
export const markAllNotificationsRead = () => apiRequest(endpoints.notificationsReadAll, { method: "POST" });

export const getBlocked = () => apiRequest<{ blocked: BlockedUser[] }>(endpoints.blocks).then((r) => r.blocked);
export const blockUser = (userId: string) => apiRequest(endpoints.blocks, { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => apiRequest(endpoints.unblock(userId), { method: "DELETE" });
export const reportUser = (userId: string, reason: ReportReason, details: string, alsoBlock: boolean) =>
  apiRequest(endpoints.reports, { method: "POST", body: { userId, reason, details, alsoBlock } });

export const getWsTicket = () => apiRequest<{ ticket: string }>(endpoints.wsTicket, { method: "POST" }).then((r) => r.ticket);

export type PickedImage = { uri: string; mimeType?: string | null; fileName?: string | null };

async function toFormData(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  const type = image.mimeType === "image/png" ? "image/png" : "image/jpeg";
  const name = image.fileName ?? (type === "image/png" ? "photo.png" : "photo.jpg");
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("file", blob, name);
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor instead of a Blob.
    form.append("file", { uri: image.uri, name, type } as unknown as Blob);
  }
  return form;
}

export async function uploadPhoto(image: PickedImage): Promise<Photo[]> {
  const res = await apiRequest<{ photos: Photo[] }>(endpoints.photos, { method: "POST", form: await toFormData(image) });
  return res.photos;
}

export async function replacePhoto(photoId: string, image: PickedImage): Promise<Photo[]> {
  const res = await apiRequest<{ photos: Photo[] }>(endpoints.photo(photoId), { method: "PUT", form: await toFormData(image) });
  return res.photos;
}

export const deletePhoto = (photoId: string) => apiRequest<{ photos: Photo[] }>(endpoints.photo(photoId), { method: "DELETE" }).then((r) => r.photos);

export const reorderPhotos = (ids: string[]) => apiRequest<{ photos: Photo[] }>(endpoints.photosOrder, { method: "PUT", body: { ids } }).then((r) => r.photos);
