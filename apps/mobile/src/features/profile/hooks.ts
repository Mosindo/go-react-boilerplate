import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { photosApi, profileApi, type PreferencesUpdate, type ProfileUpdate } from "../../api/profile";
import { queryKeys } from "../../api/queryKeys";
import type { MeProfile, MissingField, Photo } from "../../api/types";

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: profileApi.me });
}

export function useInterests() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: async () => (await profileApi.interests()).interests,
    staleTime: 24 * 3600_000
  });
}

function useProfileMutation<V>(fn: (vars: V) => Promise<MeProfile>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.myProfile, data);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discovery });
    }
  });
}

export const useUpdateProfile = () => useProfileMutation((u: ProfileUpdate) => profileApi.update(u));
export const useUpdatePreferences = () => useProfileMutation((p: PreferencesUpdate) => profileApi.updatePreferences(p));
export const useUpdatePrivacy = () =>
  useProfileMutation((v: { isPaused: boolean; hideDistance: boolean }) => profileApi.updatePrivacy(v.isPaused, v.hideDistance));
export const useUpdateLocation = () =>
  useProfileMutation((v: { latitude: number; longitude: number; city: string }) =>
    profileApi.updateLocation(v.latitude, v.longitude, v.city)
  );
export const useClearLocation = () => useProfileMutation(() => profileApi.clearLocation());

/** Photo mutations write the returned list back into the cached profile. */
export function usePhotoMutations() {
  const queryClient = useQueryClient();
  const apply = ({ photos }: { photos: Photo[] }) => {
    queryClient.setQueryData<MeProfile>(queryKeys.myProfile, (data) => {
      if (!data) return data;
      const missing: MissingField[] = data.completeness.missing.filter((m) => m !== "photo");
      if (photos.length === 0) missing.push("photo");
      return {
        ...data,
        profile: { ...data.profile, photos },
        completeness: { isComplete: missing.length === 0, missing }
      };
    });
  };
  return {
    upload: useMutation({ mutationFn: (uri: string) => photosApi.upload(uri), onSuccess: apply }),
    replace: useMutation({ mutationFn: (v: { id: string; uri: string }) => photosApi.replace(v.id, v.uri), onSuccess: apply }),
    remove: useMutation({ mutationFn: (id: string) => photosApi.remove(id), onSuccess: apply }),
    reorder: useMutation({ mutationFn: (ids: string[]) => photosApi.reorder(ids), onSuccess: apply })
  };
}

export function profileToUpdate(profile: MeProfile["profile"]): ProfileUpdate {
  return {
    firstName: profile.firstName,
    birthdate: profile.birthdate ?? "",
    gender: profile.gender ?? "",
    bio: profile.bio,
    jobTitle: profile.jobTitle,
    heightCm: profile.heightCm,
    city: profile.city,
    interestIds: profile.interests.map((i) => i.id)
  };
}
