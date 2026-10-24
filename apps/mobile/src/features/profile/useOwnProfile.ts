import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { OwnProfile } from "../../api/types";
import { queryKeys } from "../../realtime/cache";

/** The caller's profile; `data` is null (not an error) while onboarding has not created it yet. */
export function useOwnProfile() {
  return useQuery<OwnProfile | null>({
    queryKey: queryKeys.ownProfile,
    queryFn: async () => {
      try {
        return await profileApi.own();
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null;
        throw e;
      }
    },
  });
}
