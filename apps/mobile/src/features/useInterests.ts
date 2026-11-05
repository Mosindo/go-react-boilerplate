import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { queryKeys } from "../lib/queryKeys";

export function useInterests() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: profileApi.interests,
    staleTime: 60 * 60 * 1000
  });
}
