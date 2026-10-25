import { http } from "./client";
import type { MatchItem, Page } from "./types";

export const matchesApi = {
  list: (cursor?: string) => http.get<Page<MatchItem>>("/matches", { limit: 20, cursor }),
  unmatch: (matchId: string) => http.delete(`/matches/${encodeURIComponent(matchId)}`)
};
