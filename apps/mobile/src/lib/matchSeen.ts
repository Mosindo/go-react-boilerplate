// Matches the user just made by swiping are announced by the match modal. The
// realtime "match" event arrives for both people, so the toast is skipped for
// matches that were already announced locally.
const announced = new Set<string>();

export function markMatchAnnounced(conversationId: string): void {
  announced.add(conversationId);
}

export function wasMatchAnnounced(conversationId: string): boolean {
  return announced.has(conversationId);
}
