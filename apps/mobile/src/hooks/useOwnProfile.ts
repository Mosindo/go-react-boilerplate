import { useQuery } from "@tanstack/react-query";
import { getOwnProfile } from "../api/dating";
import { queryKeys } from "../lib/queryKeys";

export function useOwnProfile() {
  return useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
}
