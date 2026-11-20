import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { MyProfile } from "../../api/types";
import { keys } from "../../providers/queryClient";

/** The signed-in member's own profile; null while it has not been created. */
export function useMyProfile() {
  return useQuery<MyProfile | null>({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await profileApi.mine();
      } catch (error) {
        if (error instanceof ApiError && error.code === "profile_not_found") {
          return null;
        }
        throw error;
      }
    }
  });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}
