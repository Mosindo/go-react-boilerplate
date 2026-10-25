import type { ConversationItem, MatchItem } from "../../api";

/** Matches nobody has written to yet (the API only lists conversations holding a message): shown in the "new matches" strip. */
export function selectNewMatches(matches: MatchItem[], conversations: ConversationItem[]): MatchItem[] {
  const withMessages = new Set(conversations.map((item) => item.id));
  return matches.filter((match) => !withMessages.has(match.conversationId));
}
