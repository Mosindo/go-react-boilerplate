import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  AppNotification,
  BlockedUser,
  ChatMessage,
  Conversation,
  Interest,
  Match,
  OwnProfile,
  Preferences,
  ProfileInput,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

function withQuery(path: string, params: Record<string, string | number | undefined | null>): string {
  const query = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null && value !== "")
    .map(([key, value]) => `${key}=${encodeURIComponent(String(value))}`)
    .join("&");
  return query ? `${path}?${query}` : path;
}

// Profile ---------------------------------------------------------------

export async function fetchInterests(): Promise<Interest[]> {
  return (await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests)).interests;
}

export function fetchOwnProfile(): Promise<OwnProfile> {
  return apiRequest<OwnProfile>(endpoints.profile.own);
}

export function saveProfile(input: ProfileInput): Promise<OwnProfile> {
  return apiRequest<OwnProfile>(endpoints.profile.own, { method: "PUT", json: input });
}

export function savePreferences(input: Preferences): Promise<OwnProfile> {
  return apiRequest<OwnProfile>(endpoints.profile.preferences, { method: "PUT", json: input });
}

export function saveLocation(coords: { latitude: number; longitude: number } | null): Promise<OwnProfile> {
  return apiRequest<OwnProfile>(endpoints.profile.location, { method: "PUT", json: coords ?? {} });
}

export async function fetchPublicProfile(userId: string): Promise<PublicProfile> {
  return (await apiRequest<{ profile: PublicProfile }>(endpoints.profile.public(userId))).profile;
}

// Photos ----------------------------------------------------------------

export type UploadFile = { uri: string; name: string; type: string };

async function toFormData(file: UploadFile): Promise<FormData> {
  const form = new FormData();
  if (file.uri.startsWith("blob:") || file.uri.startsWith("data:")) {
    // Web: the picker returns an object/data URL.
    const blob = await (await fetch(file.uri)).blob();
    form.append("file", blob, file.name);
  } else {
    // React Native: the form-data shim accepts { uri, name, type }.
    form.append("file", { uri: file.uri, name: file.name, type: file.type } as unknown as Blob);
  }
  return form;
}

export async function uploadPhoto(file: UploadFile): Promise<void> {
  await apiRequest(endpoints.photos.list, { method: "POST", form: await toFormData(file), timeoutMs: 60_000 });
}

export async function replacePhoto(photoId: string, file: UploadFile): Promise<void> {
  await apiRequest(endpoints.photos.item(photoId), { method: "PUT", form: await toFormData(file), timeoutMs: 60_000 });
}

export function deletePhoto(photoId: string): Promise<void> {
  return apiRequest(endpoints.photos.item(photoId), { method: "DELETE" });
}

export async function reorderPhotos(ids: string[]): Promise<void> {
  await apiRequest(endpoints.photos.order, { method: "PUT", json: { ids } });
}

// Discovery & matching ----------------------------------------------------

export async function fetchDiscover(limit = 10): Promise<PublicProfile[]> {
  return (await apiRequest<{ profiles: PublicProfile[] }>(withQuery(endpoints.discover, { limit }))).profiles;
}

export function swipe(userId: string, action: SwipeAction): Promise<SwipeResult> {
  return apiRequest<SwipeResult>(endpoints.likes, { method: "POST", json: { userId, action } });
}

export async function fetchMatches(): Promise<Match[]> {
  return (await apiRequest<{ matches: Match[] }>(withQuery(endpoints.matches.list, { limit: 50 }))).matches;
}

export function unmatch(matchId: string): Promise<void> {
  return apiRequest(endpoints.matches.item(matchId), { method: "DELETE" });
}

// Chat ----------------------------------------------------------------------

export async function fetchConversations(): Promise<Conversation[]> {
  return (await apiRequest<{ conversations: Conversation[] }>(withQuery(endpoints.conversations.list, { limit: 50 }))).conversations;
}

export type MessagePage = { messages: ChatMessage[]; nextBefore: number | null };

export function fetchMessages(conversationId: string, before?: number | null): Promise<MessagePage> {
  return apiRequest<MessagePage>(withQuery(endpoints.conversations.messages(conversationId), { limit: 30, before }));
}

export function sendMessage(conversationId: string, body: string): Promise<ChatMessage> {
  return apiRequest<ChatMessage>(endpoints.conversations.messages(conversationId), { method: "POST", json: { body } });
}

export async function markConversationRead(conversationId: string): Promise<void> {
  await apiRequest(endpoints.conversations.read(conversationId), { method: "POST" });
}

export function hideConversation(conversationId: string): Promise<void> {
  return apiRequest(endpoints.conversations.item(conversationId), { method: "DELETE" });
}

// Notifications ---------------------------------------------------------------

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export function fetchNotifications(): Promise<NotificationsPage> {
  return apiRequest<NotificationsPage>(withQuery(endpoints.notifications.list, { limit: 50 }));
}

export function markNotificationRead(id: string): Promise<void> {
  return apiRequest(endpoints.notifications.read(id), { method: "POST" });
}

export function markAllNotificationsRead(): Promise<void> {
  return apiRequest(endpoints.notifications.readAll, { method: "POST" });
}

// Safety ------------------------------------------------------------------------

export function blockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.safety.blocks, { method: "POST", json: { userId } });
}

export function unblockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.safety.unblock(userId), { method: "DELETE" });
}

export async function fetchBlocked(): Promise<BlockedUser[]> {
  return (await apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks)).blocks;
}

export function reportUser(userId: string, reason: ReportReason, details: string, alsoBlock: boolean): Promise<void> {
  return apiRequest(endpoints.safety.reports, { method: "POST", json: { userId, reason, details, alsoBlock } });
}
