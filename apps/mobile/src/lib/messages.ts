import type { Message, MessagesPage } from "../api/types";

/**
 * Inserts a message at the head of the first page (messages are newest-first),
 * ignoring duplicates so that the REST response and the WebSocket echo of the
 * same message never render twice.
 */
export function prependMessage(pages: MessagesPage[], message: Message): MessagesPage[] {
  if (pages.some((page) => page.messages.some((m) => m.id === message.id))) {
    return pages;
  }
  if (pages.length === 0) {
    return [{ messages: [message] }];
  }
  const [first, ...rest] = pages;
  return [{ ...first, messages: [message, ...first.messages] }, ...rest];
}

export function flattenMessages(pages: MessagesPage[] | undefined): Message[] {
  return pages ? pages.flatMap((page) => page.messages) : [];
}
