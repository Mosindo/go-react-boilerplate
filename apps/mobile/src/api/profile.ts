import { http } from "./client";
import type {
  Interest,
  LocationInput,
  MyProfile,
  Preferences,
  ProfileInput,
  ProfileState,
  PublicProfile
} from "./types";

export const profileApi = {
  get: () => http.get<ProfileState>("/profile"),
  save: (input: ProfileInput) => http.put<MyProfile>("/profile", input),
  savePreferences: (input: Preferences) => http.put<Preferences>("/preferences", input),
  setLocation: (input: LocationInput) => http.put<MyProfile>("/profile/location", input),
  clearLocation: () => http.delete("/profile/location"),
  interests: async () => (await http.get<{ items: Interest[] }>("/interests")).items,
  publicProfile: (userId: string) => http.get<PublicProfile>(`/profiles/${encodeURIComponent(userId)}`)
};
