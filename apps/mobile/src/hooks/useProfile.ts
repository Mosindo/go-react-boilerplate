import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getInterests,
  deletePhoto,
  reorderPhotos,
  saveLocation,
  saveProfile,
  savePreferences,
  uploadPhoto,
  type UploadFile
} from "../api/profile";
import { getCandidateProfile } from "../api/discover";
import { queryKeys } from "../api/queryKeys";
import type { LocationInput, Preferences, ProfileInput } from "../api/types";

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: getInterests, staleTime: 24 * 60 * 60 * 1000 });
}

/** Wraps a profile-affecting call and refreshes /me (profileComplete is computed by the server). */
function useMeMutation<TInput, TResult>(fn: (input: TInput) => Promise<TResult>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.me })
  });
}

export const useSaveProfile = () => useMeMutation((input: ProfileInput) => saveProfile(input));
export const useSaveLocation = () => useMeMutation((input: LocationInput) => saveLocation(input));
export const useSavePreferences = () => useMeMutation((input: Preferences) => savePreferences(input));
export const useUploadPhoto = () => useMeMutation((file: UploadFile) => uploadPhoto(file));
export const useDeletePhoto = () => useMeMutation((photoId: string) => deletePhoto(photoId));
export const useReorderPhotos = () => useMeMutation((photoIds: string[]) => reorderPhotos(photoIds));

export function useCandidateProfile(userId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.candidate(userId ?? ""),
    queryFn: () => getCandidateProfile(userId ?? ""),
    enabled: enabled && Boolean(userId)
  });
}
