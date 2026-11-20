// Likes sent from this device. When the match is created, the realtime event
// can arrive before the HTTP response: knowing which profile we just liked
// lets us skip the toast because the match modal already celebrates it.
const pendingLikes = new Map<string, ReturnType<typeof setTimeout>>();

export const matchCelebrations = {
  markLike(userId: string) {
    const existing = pendingLikes.get(userId);
    if (existing) {
      clearTimeout(existing);
    }
    pendingLikes.set(
      userId,
      setTimeout(() => pendingLikes.delete(userId), 15_000)
    );
  },
  /** True when the match was triggered by a like sent from this device. */
  consume(userId: string): boolean {
    const timer = pendingLikes.get(userId);
    if (!timer) {
      return false;
    }
    clearTimeout(timer);
    pendingLikes.delete(userId);
    return true;
  }
};
