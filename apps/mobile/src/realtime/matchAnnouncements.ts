// Matches already celebrated in-app (e.g. by the swipe result modal), so the
// realtime echo of the same match does not show a second toast.
const announced = new Set<string>();

export function announceMatch(matchId: string) {
  announced.add(matchId);
}

export function wasMatchAnnounced(matchId: string): boolean {
  return announced.has(matchId);
}
