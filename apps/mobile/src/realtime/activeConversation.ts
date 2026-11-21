// Tracks which conversation is on screen so incoming messages for it don't
// trigger a toast and are marked as read immediately.
let activeUserId: string | null = null;

export function setActiveConversation(userId: string | null): void {
  activeUserId = userId;
}

export function getActiveConversation(): string | null {
  return activeUserId;
}
