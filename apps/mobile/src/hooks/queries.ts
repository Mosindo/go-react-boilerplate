import { useQuery } from "@tanstack/react-query";
import { listInterests, listMatches } from "../api/dating";
import { listNotifications } from "../api/platform";
import type { Interest, MatchesPage, NotificationsPage } from "../api/types";
import { useAuth } from "./useAuth";

/** Conversations + new matches. Kept fresh by realtime events (invalidated in RealtimeProvider). */
export function useMatches() {
  const { status } = useAuth();
  return useQuery<MatchesPage, Error>({
    queryKey: ["matches"],
    queryFn: () => listMatches(),
    enabled: status === "signedIn",
    staleTime: 10_000
  });
}

export function useNotifications() {
  const { status } = useAuth();
  return useQuery<NotificationsPage, Error>({
    queryKey: ["notifications"],
    queryFn: () => listNotifications(),
    enabled: status === "signedIn",
    staleTime: 10_000
  });
}

export function useInterestCatalog() {
  return useQuery<Interest[], Error>({
    queryKey: ["interests"],
    queryFn: listInterests,
    staleTime: Infinity
  });
}
