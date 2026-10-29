import { useQuery } from "@tanstack/react-query";
import * as profileApi from "../api/profile";
import type { Interest, OwnProfile, Photo, Preferences } from "../api/types";

export const queryKeys = {
  profile: ["profile"] as const,
  preferences: ["preferences"] as const,
  photos: ["photos"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  conversations: ["conversations"] as const,
  notifications: ["notifications"] as const,
  blocked: ["blocked"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  publicProfile: (userId: string) => ["publicProfile", userId] as const
};

export const useProfile = () =>
  useQuery<OwnProfile>({ queryKey: queryKeys.profile, queryFn: profileApi.fetchProfile });

export const usePreferences = () =>
  useQuery<Preferences>({ queryKey: queryKeys.preferences, queryFn: profileApi.fetchPreferences });

export const usePhotos = () =>
  useQuery<Photo[]>({ queryKey: queryKeys.photos, queryFn: profileApi.fetchPhotos });

export const useInterests = () =>
  useQuery<Interest[]>({
    queryKey: queryKeys.interests,
    queryFn: profileApi.fetchInterests,
    staleTime: Infinity
  });
