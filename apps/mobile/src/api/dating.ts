import { Platform } from "react-native";
import { apiRequest } from "./client";
import type {
  AppNotification,
  BlockedUser,
  Card,
  ChatMessage,
  Gender,
  Interest,
  MatchSummary,
  MessagePage,
  MyProfile,
  NotificationPage,
  Preferences,
  ProfileInput,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

export const datingApi = {
  interests: async (): Promise<Interest[]> => (await apiRequest<{ items: Interest[] }>("/interests")).items,

  myProfile: (): Promise<MyProfile> => apiRequest<MyProfile>("/me/profile"),
  saveProfile: (input: ProfileInput): Promise<MyProfile> => apiRequest<MyProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (input: Preferences): Promise<MyProfile> => apiRequest<MyProfile>("/me/preferences", { method: "PUT", body: input }),
  saveLocation: (latitude: number, longitude: number, city?: string): Promise<MyProfile> =>
    apiRequest<MyProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  savePrivacy: (input: { isVisible?: boolean; showDistance?: boolean }): Promise<MyProfile> =>
    apiRequest<MyProfile>("/me/privacy", { method: "PUT", body: input }),
  profile: (userId: string): Promise<Card> => apiRequest<Card>(`/profiles/${userId}`),

  uploadPhoto: async (file: PhotoFile): Promise<{ id: string }> =>
    apiRequest<{ id: string }>("/me/photos", { method: "POST", formData: await photoForm(file) }),
  replacePhoto: async (id: string, file: PhotoFile): Promise<void> =>
    apiRequest(`/me/photos/${id}`, { method: "PUT", formData: await photoForm(file) }),
  deletePhoto: (id: string): Promise<void> => apiRequest(`/me/photos/${id}`, { method: "DELETE" }),
  reorderPhotos: (ids: string[]): Promise<void> => apiRequest("/me/photos/order", { method: "PUT", body: { ids } }),

  discover: async (limit = 10): Promise<Card[]> => (await apiRequest<{ items: Card[] }>(`/discover?limit=${limit}`)).items,
  swipe: (targetId: string, action: SwipeAction): Promise<SwipeResult> =>
    apiRequest<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } }),

  matches: async (): Promise<MatchSummary[]> => (await apiRequest<{ items: MatchSummary[] }>("/matches?limit=50")).items,
  unmatch: (matchId: string): Promise<void> => apiRequest(`/matches/${matchId}`, { method: "DELETE" }),

  messages: (matchId: string, before?: string): Promise<MessagePage> =>
    apiRequest<MessagePage>(`/matches/${matchId}/messages?limit=30${before ? `&before=${before}` : ""}`),
  sendMessage: (matchId: string, body: string): Promise<ChatMessage> =>
    apiRequest<ChatMessage>(`/matches/${matchId}/messages`, { method: "POST", body: { body } }),
  markRead: (matchId: string): Promise<void> => apiRequest(`/matches/${matchId}/read`, { method: "POST" }),
  clearConversation: (matchId: string): Promise<void> => apiRequest(`/matches/${matchId}/messages`, { method: "DELETE" }),

  notifications: (): Promise<NotificationPage> => apiRequest<NotificationPage>("/notifications?limit=50"),
  markNotificationRead: (id: string): Promise<void> => apiRequest(`/notifications/${id}/read`, { method: "POST" }),
  markAllNotificationsRead: (): Promise<void> => apiRequest("/notifications/read-all", { method: "POST" }),

  blocks: async (): Promise<BlockedUser[]> => (await apiRequest<{ items: BlockedUser[] }>("/blocks")).items,
  block: (userId: string): Promise<void> => apiRequest("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string): Promise<void> => apiRequest(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, alsoBlock: boolean): Promise<void> =>
    apiRequest("/reports", { method: "POST", body: { userId, reason, details, alsoBlock } })
};

export type PhotoFile = { uri: string; name?: string; mimeType?: string };

async function photoForm(file: PhotoFile): Promise<FormData> {
  const form = new FormData();
  const name = file.name ?? "photo.jpg";
  if (Platform.OS === "web") {
    // Browsers need a real Blob; the picker returns a blob:/data: URI.
    form.append("file", await (await fetch(file.uri)).blob(), name);
  } else {
    // React Native's FormData accepts { uri, name, type } descriptors for files.
    form.append("file", { uri: file.uri, name, type: file.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  non_binary: "Non-binaire"
};
export type { AppNotification };
