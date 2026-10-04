import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Platform } from "react-native";
import * as profileApi from "../api/profile";
import { queryKeys } from "../api/queryKeys";
import type { OwnProfile, Photo, Preferences, ProfileInput } from "../api/types";
import type { PhotoFile } from "../api/profile";
import { bumpDiscoverEpoch } from "../store/deckIntent";

/** Own profile; `null` data means the profile was never created (onboarding needed). */
export function useOwnProfile(enabled = true) {
  return useQuery({
    queryKey: queryKeys.profile,
    queryFn: profileApi.getOwnProfile,
    enabled,
    staleTime: 30_000
  });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.listInterests, staleTime: Infinity });
}

export function usePublicProfile(userId: string) {
  return useQuery({
    queryKey: queryKeys.publicProfile(userId),
    queryFn: () => profileApi.getPublicProfile(userId),
    staleTime: 60_000
  });
}

export function useSaveProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: ProfileInput) => profileApi.saveProfile(input),
    onSuccess: (profile) => {
      client.setQueryData<OwnProfile | null>(queryKeys.profile, profile);
    }
  });
}

export function useSavePreferences() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (preferences: Preferences) => profileApi.savePreferences(preferences),
    onSuccess: (preferences) => {
      client.setQueryData<OwnProfile | null>(queryKeys.profile, (profile) =>
        profile ? { ...profile, preferences } : profile
      );
      bumpDiscoverEpoch();
    }
  });
}

function patchPhotos(client: ReturnType<typeof useQueryClient>, update: (photos: Photo[]) => Photo[]) {
  client.setQueryData<OwnProfile | null>(queryKeys.profile, (profile) =>
    profile ? { ...profile, photos: update(profile.photos) } : profile
  );
}

export function useUploadPhoto() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (file: PhotoFile) => profileApi.uploadPhoto(file, Platform.OS),
    onSuccess: (photo) => patchPhotos(client, (photos) => [...photos.filter((p) => p.id !== photo.id), photo])
  });
}

export function useDeletePhoto() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => profileApi.deletePhoto(id),
    onSuccess: (_void, id) =>
      patchPhotos(client, (photos) => photos.filter((p) => p.id !== id).map((p, index) => ({ ...p, position: index }))),
    onSettled: () => {
      void client.invalidateQueries({ queryKey: queryKeys.profile });
    }
  });
}

export function useReorderPhotos() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => profileApi.reorderPhotos(ids),
    onSuccess: (photos) => patchPhotos(client, () => photos)
  });
}

export function useSetLocation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ latitude, longitude }: { latitude: number; longitude: number }) =>
      profileApi.setLocation(latitude, longitude),
    onSuccess: () => {
      client.setQueryData<OwnProfile | null>(queryKeys.profile, (p) => (p ? { ...p, hasLocation: true } : p));
      bumpDiscoverEpoch();
    }
  });
}

export function useClearLocation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: profileApi.clearLocation,
    onSuccess: () => {
      client.setQueryData<OwnProfile | null>(queryKeys.profile, (p) => (p ? { ...p, hasLocation: false } : p));
    }
  });
}
