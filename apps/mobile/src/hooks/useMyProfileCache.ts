import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { MyProfileResponse, PhotoView } from "../api/types";
import { queryKeys } from "./queries";

/** Applies a fresh photo list to the cached "me" profile so every screen updates at once. */
export function useSetPhotosInCache() {
  const queryClient = useQueryClient();
  return useCallback(
    (photos: PhotoView[]) => {
      queryClient.setQueryData<MyProfileResponse>(queryKeys.myProfile, (old) => {
        if (!old) {
          return old;
        }
        const onboarding = { ...old.onboarding, photos: photos.length > 0 };
        onboarding.complete = onboarding.profile && onboarding.preferences && onboarding.photos;
        return { ...old, onboarding, profile: old.profile ? { ...old.profile, photos } : null };
      });
    },
    [queryClient]
  );
}
