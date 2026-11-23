/**
 * Tiny in-memory registry used to avoid redundant in-app toasts: no "new
 * message" toast for the conversation on screen, no "new match" toast for a
 * match already celebrated full-screen.
 */
let activeConversationId: string | null = null;
const celebratedMatches = new Set<string>();

export function setActiveConversation(conversationId: string | null) {
  activeConversationId = conversationId;
}

export function isConversationActive(conversationId: string | undefined): boolean {
  return conversationId !== undefined && conversationId === activeConversationId;
}

export function markMatchCelebrated(matchId: string) {
  celebratedMatches.add(matchId);
}

export function wasMatchCelebrated(matchId: string): boolean {
  return celebratedMatches.has(matchId);
}
