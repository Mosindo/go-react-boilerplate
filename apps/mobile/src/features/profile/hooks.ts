import { useQuery } from '@tanstack/react-query';
import { isNotFound, profileApi, queryKeys } from '../../api';

/** Own profile. A 404 (`profile_not_found`) is a normal state during onboarding, not retried. */
export function useOwnProfile() {
  return useQuery({
    queryKey: queryKeys.ownProfile,
    queryFn: () => profileApi.own(),
    retry: (count, error) => !isNotFound(error) && count < 2,
  });
}

export function useInterests() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: () => profileApi.interests(),
    staleTime: Infinity,
  });
}

export function usePreferences() {
  return useQuery({ queryKey: queryKeys.preferences, queryFn: () => profileApi.preferences() });
}

export function usePrivacy() {
  return useQuery({ queryKey: queryKeys.privacy, queryFn: () => profileApi.privacy() });
}
