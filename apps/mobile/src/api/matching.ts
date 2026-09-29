import type { ConversationSummary } from "./models";
import { apiRequest } from "./client";
import type { SwipeAction } from "../lib/dating/swipe";

export type SwipeResult = { matched: boolean; conversation: ConversationSummary | null };

export async function swipe(userId: string, action: SwipeAction): Promise<SwipeResult> {
  const res = await apiRequest<SwipeResult>("/swipes", {
    method: "POST",
    body: JSON.stringify({ userId, action })
  });
  return { matched: Boolean(res?.matched), conversation: res?.conversation ?? null };
}

export async function unmatch(matchId: string): Promise<void> {
  await apiRequest<void>(`/matches/${encodeURIComponent(matchId)}`, { method: "DELETE" });
}
