import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { fetchInterests, fetchPhotos, fetchPreferences, fetchProfile } from "../api/profile";

/** The profile of the signed-in user; resolves to null before onboarding created it. */
export function useProfileQuery() {
  return useQuery({
    queryKey: ["profile"],
    queryFn: async () => {
      try {
        return await fetchProfile();
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) {
          return null;
        }
        throw err;
      }
    }
  });
}

export function usePreferencesQuery() {
  return useQuery({
    queryKey: ["preferences"],
    queryFn: async () => {
      try {
        return await fetchPreferences();
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) {
          return null;
        }
        throw err;
      }
    }
  });
}

export function useInterestsQuery() {
  return useQuery({ queryKey: ["interests"], queryFn: fetchInterests, staleTime: 60 * 60 * 1000 });
}

export function usePhotosQuery() {
  return useQuery({ queryKey: ["photos"], queryFn: fetchPhotos });
}
