// Ids already reacted to during this session. The server never returns them again once the swipe
// is stored; this set only covers the short window while a swipe request is still in flight.
const swiped = new Set<string>();

export const markSwiped = (id: string) => swiped.add(id);
export const unmarkSwiped = (id: string) => swiped.delete(id);
export const wasSwiped = (id: string) => swiped.has(id);
export const resetSwipedMemory = () => swiped.clear();
