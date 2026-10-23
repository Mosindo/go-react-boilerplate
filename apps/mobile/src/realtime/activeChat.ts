let activeMatchId: string | null = null;
const removedListeners = new Set<(matchId: string) => void>();

/**
 * Tracks which conversation is on screen so incoming messages don't raise its unread count,
 * and lets the chat screen leave when its match is removed remotely.
 */
export const activeChat = {
  get: () => activeMatchId,
  set: (matchId: string | null) => {
    activeMatchId = matchId;
  },
  onRemoved: (listener: (matchId: string) => void) => {
    removedListeners.add(listener);
    return () => {
      removedListeners.delete(listener);
    };
  },
  emitRemoved: (matchId: string) => removedListeners.forEach((listener) => listener(matchId))
};
