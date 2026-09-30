import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../lib/api/endpoints";
import type { OwnProfile, ProfileField } from "../../lib/api/types";
import { queryKeys } from "../../lib/queryClient";

export function useProfile() {
  return useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.get });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

/** Wraps a profile-returning mutation and writes the result into the cache. */
export function useProfileMutation<TVars>(fn: (vars: TVars) => Promise<OwnProfile>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (profile) => {
      client.setQueryData(queryKeys.profile, profile);
      void client.invalidateQueries({ queryKey: queryKeys.discovery });
    }
  });
}

/** Updates only the photos of the cached profile. */
export function usePhotosCacheUpdater() {
  const client = useQueryClient();
  return (photos: OwnProfile["photos"]) => {
    client.setQueryData<OwnProfile>(queryKeys.profile, (profile) => {
      if (!profile) return profile;
      const missing: ProfileField[] = profile.completeness.missing.filter((m) => m !== "photos");
      if (photos.length === 0) missing.push("photos");
      return { ...profile, photos, completeness: { complete: missing.length === 0, missing } };
    });
  };
}
