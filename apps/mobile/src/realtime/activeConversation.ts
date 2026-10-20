let activeConversationId: string | null = null;

export function setActiveConversation(id: string | null): void {
  activeConversationId = id;
}

export function getActiveConversation(): string | null {
  return activeConversationId;
}
