import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { fetchInterests, fetchMyProfile } from "../../api/profile";
import { queryKeys } from "../../api/keys";

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.myProfile, queryFn: fetchMyProfile });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: fetchInterests, staleTime: Infinity });
}

export async function invalidateProfileData(qc: QueryClient): Promise<void> {
  await Promise.all([
    qc.invalidateQueries({ queryKey: queryKeys.myProfile }),
    qc.invalidateQueries({ queryKey: queryKeys.me }),
    qc.invalidateQueries({ queryKey: ["discover"] })
  ]);
}

export function useProfileInvalidation(): () => Promise<void> {
  const qc = useQueryClient();
  return () => invalidateProfileData(qc);
}
