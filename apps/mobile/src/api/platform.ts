// Typed wrappers around the server's resource endpoints. Screens never call fetch directly.
import { Platform } from "react-native";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  AppNotification,
  BlockedUser,
  Conversation,
  Interest,
  MeProfile,
  Match,
  Message,
  MessagesPage,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

export function getMyProfile(): Promise<MeProfile> {
  return apiRequest<MeProfile>(endpoints.profile.me);
}

export function saveProfile(input: ProfileInput): Promise<MeProfile> {
  return apiRequest<MeProfile>(endpoints.profile.me, { method: "PUT", body: JSON.stringify(input) });
}

export function savePreferences(input: Preferences): Promise<MeProfile> {
  return apiRequest<MeProfile>(endpoints.profile.preferences, { method: "PUT", body: JSON.stringify(input) });
}

export function savePrivacy(input: { discoverable?: boolean; showDistance?: boolean }): Promise<MeProfile> {
  return apiRequest<MeProfile>(endpoints.profile.privacy, { method: "PATCH", body: JSON.stringify(input) });
}

export async function listInterests(): Promise<Interest[]> {
  const res = await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests);
  return res.interests;
}

export function getProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(endpoints.profile.detail(userId));
}

export type UploadableImage = { uri: string; name?: string; mimeType?: string };

async function imageForm(image: UploadableImage): Promise<FormData> {
  const form = new FormData();
  const name = image.name ?? "photo.jpg";
  if (Platform.OS === "web") {
    // Browsers need a real Blob; the picker hands back a blob:/data: URI.
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, name);
  } else {
    // React Native's FormData accepts { uri, name, type } objects in place of Blobs.
    form.append("photo", { uri: image.uri, name, type: image.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export async function uploadPhoto(image: UploadableImage): Promise<{ id: string; url: string }> {
  return apiRequest(endpoints.photos.upload, { method: "POST", body: await imageForm(image), timeoutMs: 45_000 });
}

export async function replacePhoto(photoId: string, image: UploadableImage): Promise<{ id: string; url: string }> {
  return apiRequest(endpoints.photos.item(photoId), { method: "PUT", body: await imageForm(image), timeoutMs: 45_000 });
}

export function deletePhoto(photoId: string): Promise<void> {
  return apiRequest(endpoints.photos.item(photoId), { method: "DELETE" });
}

export function makeMainPhoto(photoId: string): Promise<void> {
  return apiRequest(endpoints.photos.main(photoId), { method: "PUT" });
}

export function reorderPhotos(photoIds: string[]): Promise<void> {
  return apiRequest(endpoints.photos.order, { method: "PUT", body: JSON.stringify({ photoIds }) });
}

export async function getDiscover(limit = 10): Promise<PublicProfile[]> {
  const res = await apiRequest<{ profiles: PublicProfile[] }>(`${endpoints.discover}?limit=${limit}`);
  return res.profiles;
}

export function swipe(targetId: string, action: SwipeAction): Promise<SwipeResult> {
  return apiRequest<SwipeResult>(endpoints.swipes, { method: "POST", body: JSON.stringify({ targetId, action }) });
}

export async function listMatches(limit = 50, offset = 0): Promise<Match[]> {
  const res = await apiRequest<{ matches: Match[] }>(`${endpoints.matches.list}?limit=${limit}&offset=${offset}`);
  return res.matches;
}

export function listLikesReceived(limit = 30, offset = 0): Promise<{ profiles: PublicProfile[]; total: number }> {
  return apiRequest(`${endpoints.matches.likesReceived}?limit=${limit}&offset=${offset}`);
}

export function unmatch(matchId: string): Promise<void> {
  return apiRequest(endpoints.matches.item(matchId), { method: "DELETE" });
}

export async function listConversations(limit = 50, offset = 0): Promise<Conversation[]> {
  const res = await apiRequest<{ conversations: Conversation[] }>(`${endpoints.chat.conversations}?limit=${limit}&offset=${offset}`);
  return res.conversations;
}

export async function getUnreadCount(): Promise<number> {
  const res = await apiRequest<{ unread: number }>(endpoints.chat.unread);
  return res.unread;
}

export function listMessages(matchId: string, before?: string, limit = 30): Promise<MessagesPage> {
  const query = `limit=${limit}${before ? `&before=${before}` : ""}`;
  return apiRequest<MessagesPage>(`${endpoints.chat.messages(matchId)}?${query}`);
}

export function sendMessage(matchId: string, body: string): Promise<Message> {
  return apiRequest<Message>(endpoints.chat.messages(matchId), { method: "POST", body: JSON.stringify({ body }) });
}

export function markConversationRead(matchId: string): Promise<void> {
  return apiRequest(endpoints.chat.read(matchId), { method: "POST" });
}

export function clearConversation(matchId: string): Promise<void> {
  return apiRequest(endpoints.chat.item(matchId), { method: "DELETE" });
}

export function blockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.safety.blocks, { method: "POST", body: JSON.stringify({ userId }) });
}

export function unblockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.safety.unblock(userId), { method: "DELETE" });
}

export async function listBlocked(): Promise<BlockedUser[]> {
  const res = await apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks);
  return res.blocks;
}

export function reportUser(userId: string, reason: ReportReason, details: string): Promise<void> {
  return apiRequest(endpoints.safety.reports, { method: "POST", body: JSON.stringify({ userId, reason, details }) });
}

export function listNotifications(limit = 50, offset = 0): Promise<{ notifications: AppNotification[]; unread: number }> {
  return apiRequest(`${endpoints.notifications.list}?limit=${limit}&offset=${offset}`);
}

export function markNotificationRead(notificationId: string): Promise<AppNotification> {
  return apiRequest(endpoints.notifications.markRead(notificationId), { method: "POST" });
}

export function markAllNotificationsRead(): Promise<void> {
  return apiRequest(endpoints.notifications.readAll, { method: "POST" });
}

export function requestRealtimeTicket(): Promise<{ ticket: string; expiresIn: number }> {
  return apiRequest(endpoints.realtime.ticket, { method: "POST", silent: true });
}
