import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  deletePhoto,
  reorderPhotos,
  uploadPhoto,
  type UploadableFile
} from "../api/photos";
import {
  getMyProfile,
  getPreferences,
  listInterests,
  saveLocation,
  saveMyProfile,
  savePreferences,
  type Interest,
  type MyProfile,
  type Preferences,
  type ProfileInput
} from "../api/profile";
import { blockUser, listBlocks, unblockUser, type BlockedUser } from "../api/safety";

export const queryKeys = {
  profile: ["me", "profile"] as const,
  preferences: ["me", "preferences"] as const,
  interests: ["interests"] as const,
  blocks: ["blocks"] as const
};

export function useMyProfile() {
  return useQuery<MyProfile | null, Error>({ queryKey: queryKeys.profile, queryFn: getMyProfile });
}

export function usePreferences(enabled = true) {
  return useQuery<Preferences | null, Error>({
    queryKey: queryKeys.preferences,
    queryFn: getPreferences,
    enabled
  });
}

export function useInterests() {
  return useQuery<Interest[], Error>({
    queryKey: queryKeys.interests,
    queryFn: listInterests,
    staleTime: 60 * 60 * 1000
  });
}

export function useBlocks() {
  return useQuery<BlockedUser[], Error>({ queryKey: queryKeys.blocks, queryFn: listBlocks });
}

export function useSaveProfile() {
  const queryClient = useQueryClient();
  return useMutation<MyProfile, Error, ProfileInput>({
    mutationFn: saveMyProfile,
    onSuccess: (profile) => {
      queryClient.setQueryData(queryKeys.profile, profile);
      // First save creates default preferences server-side.
      void queryClient.invalidateQueries({ queryKey: queryKeys.preferences });
    }
  });
}

export function useSavePreferences() {
  const queryClient = useQueryClient();
  return useMutation<Preferences, Error, Preferences>({
    mutationFn: savePreferences,
    onSuccess: (preferences) => {
      queryClient.setQueryData(queryKeys.preferences, preferences);
      // Discovery results depend on preferences.
      void queryClient.invalidateQueries({ queryKey: ["discover"] });
    }
  });
}

export function useSaveLocation() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { latitude: number; longitude: number }>({
    mutationFn: ({ latitude, longitude }) => saveLocation(latitude, longitude),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
      void queryClient.invalidateQueries({ queryKey: ["discover"] });
    }
  });
}

export function useUploadPhoto() {
  const queryClient = useQueryClient();
  return useMutation<unknown, Error, UploadableFile>({
    mutationFn: uploadPhoto,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.profile })
  });
}

export function useDeletePhoto() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: deletePhoto,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.profile })
  });
}

export function useReorderPhotos() {
  const queryClient = useQueryClient();
  return useMutation<unknown, Error, string[]>({
    mutationFn: reorderPhotos,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.profile })
  });
}

export function useBlockUser() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: blockUser,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.blocks })
  });
}

export function useUnblockUser() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: unblockUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: ["discover"] });
    }
  });
}
