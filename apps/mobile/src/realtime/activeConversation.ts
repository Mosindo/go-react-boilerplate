let activeConversationId: string | null = null;

/** The thread currently on screen: its incoming messages are read immediately and do not raise unread counters. */
export function setActiveConversation(conversationId: string | null): void {
  activeConversationId = conversationId;
}

export function getActiveConversation(): string | null {
  return activeConversationId;
}
