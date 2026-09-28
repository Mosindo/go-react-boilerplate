let openConversationId: string | null = null;

/** Tracks which chat thread is on screen so realtime events do not count its messages as unread. */
export function setOpenConversation(id: string | null): void {
  openConversationId = id;
}

export function getOpenConversation(): string | null {
  return openConversationId;
}
