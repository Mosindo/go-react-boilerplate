import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type {
  ChatMessage,
  Gender,
  Interest,
  MatchesPage,
  MessagesPage,
  Photo,
  Preferences,
  PublicProfile,
  SelfState,
  SwipeAction,
  SwipeResult
} from "./types";

export const getSelf = () => apiRequest<SelfState>(endpoints.profile);

export const saveProfile = (input: { firstName: string; birthdate: string; gender: Gender; bio: string }) =>
  apiRequest<SelfState>(endpoints.profile, { method: "PUT", body: input });

export const savePrivacy = (input: { showAge: boolean; showDistance: boolean; discoverable: boolean }) =>
  apiRequest<SelfState>(endpoints.profilePrivacy, { method: "PUT", body: input });

export const saveLocation = (input: { latitude: number; longitude: number; city: string }) =>
  apiRequest<SelfState>(endpoints.profileLocation, { method: "PUT", body: input });

export const savePreferences = (input: Preferences) => apiRequest<SelfState>(endpoints.preferences, { method: "PUT", body: input });

export const saveInterests = (interestIds: number[]) =>
  apiRequest<SelfState>(endpoints.profileInterests, { method: "PUT", body: { interestIds } });

export const listInterests = async (): Promise<Interest[]> => (await apiRequest<{ interests: Interest[] }>(endpoints.interests)).interests;

export const getProfileOf = (userId: string) => apiRequest<PublicProfile>(endpoints.profileOf(userId));

export type UploadFile = { uri: string; name: string; type: string };

export async function uploadPhoto(file: UploadFile, replacePhotoId?: string): Promise<Photo> {
  const form = new FormData();
  // React Native's FormData accepts a { uri, name, type } descriptor for files.
  form.append("file", file as unknown as Blob);
  return apiRequest<Photo>(endpoints.photos, { method: "POST", form, query: { replace: replacePhotoId } });
}

export const deletePhoto = (photoId: string) => apiRequest(endpoints.photo(photoId), { method: "DELETE" });

export const reorderPhotos = async (photoIds: string[]): Promise<Photo[]> =>
  (await apiRequest<{ photos: Photo[] }>(endpoints.photosOrder, { method: "PUT", body: { photoIds } })).photos;

export const discover = async (limit = 10): Promise<PublicProfile[]> =>
  (await apiRequest<{ profiles: PublicProfile[] }>(endpoints.discover, { query: { limit } })).profiles;

export const likesReceived = async (limit = 30, offset = 0): Promise<PublicProfile[]> =>
  (await apiRequest<{ profiles: PublicProfile[] }>(endpoints.likesReceived, { query: { limit, offset } })).profiles;

export const swipe = (userId: string, action: SwipeAction) =>
  apiRequest<SwipeResult>(endpoints.swipes, { method: "POST", body: { userId, action } });

export const listMatches = (limit = 50, offset = 0) => apiRequest<MatchesPage>(endpoints.matches, { query: { limit, offset } });

export const listMessages = (matchId: string, before?: string, limit = 40) =>
  apiRequest<MessagesPage>(endpoints.messages(matchId), { query: { before, limit } });

export const sendMessage = (matchId: string, body: string) =>
  apiRequest<ChatMessage>(endpoints.messages(matchId), { method: "POST", body: { body } });

export const markMatchRead = (matchId: string) => apiRequest(endpoints.markRead(matchId), { method: "POST" });

export const hideMatch = (matchId: string) => apiRequest(endpoints.hide(matchId), { method: "POST" });

export const unmatch = (matchId: string) => apiRequest(endpoints.match(matchId), { method: "DELETE" });
