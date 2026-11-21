import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { MeProfile, Photo } from "../../api/types";

export function useMyProfile(enabled = true) {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.me, enabled, staleTime: 60_000 });
}

export function useInterestsCatalogue() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: async () => (await profileApi.catalogue()).interests,
    staleTime: Infinity
  });
}

/** Writes a fresh /me/profile payload into the cache. */
export function useSetMyProfile() {
  const queryClient = useQueryClient();
  return (profile: MeProfile) => queryClient.setQueryData(queryKeys.myProfile, profile);
}

/** Updates only the photos of the cached profile (photo endpoints return photos). */
export function useSetMyPhotos() {
  const queryClient = useQueryClient();
  return (photos: Photo[]) =>
    queryClient.setQueryData<MeProfile>(queryKeys.myProfile, (current) => {
      if (!current) return current;
      const missing = current.missing.filter((m) => m !== "photo");
      if (photos.length === 0) missing.push("photo");
      return { ...current, photos, missing, isComplete: missing.length === 0 };
    });
}

export function useProfileMutation<TArgs>(fn: (args: TArgs) => Promise<MeProfile>) {
  const setProfile = useSetMyProfile();
  return useMutation({ mutationFn: fn, onSuccess: setProfile });
}
