import { useQuery } from "@tanstack/react-query";
import { profileApi, queryKeys } from "../../api";

export function useProfileState() {
  return useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.get, staleTime: 60_000 });
}

export function useInterestsCatalog() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: Infinity });
}
