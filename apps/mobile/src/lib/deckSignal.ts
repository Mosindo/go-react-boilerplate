// Tiny flag shared between screens: changing preferences, position or visibility makes the discovery
// queue obsolete, so the "Découvrir" tab reloads it the next time it is focused.
let stale = false;

export function markDeckStale(): void {
  stale = true;
}

/** Returns true once after markDeckStale() and resets the flag. */
export function consumeDeckStale(): boolean {
  const was = stale;
  stale = false;
  return was;
}
