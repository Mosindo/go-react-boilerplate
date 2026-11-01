import type { Message } from "../api/types";

/** Insere un message (ex: temps reel) dans une liste triee du plus recent au plus ancien, sans doublon. */
export function upsertMessage(list: Message[], message: Message): Message[] {
  if (list.some((m) => m.id === message.id)) {
    return list;
  }
  const next = [message, ...list];
  next.sort((a, b) => {
    const diff = Date.parse(b.createdAt) - Date.parse(a.createdAt);
    return diff !== 0 ? diff : b.id.localeCompare(a.id);
  });
  return next;
}

/** Fusionne des pages de messages (plus recent d'abord), en supprimant les doublons. */
export function mergePages(pages: Message[][]): Message[] {
  const seen = new Set<string>();
  const out: Message[] = [];
  for (const page of pages) {
    for (const m of page) {
      if (!seen.has(m.id)) {
        seen.add(m.id);
        out.push(m);
      }
    }
  }
  return out;
}

/** Vrai si le message envoye a ete lu par l'autre participant. */
export function isRead(message: Message, otherLastReadAt: string): boolean {
  return Date.parse(otherLastReadAt) >= Date.parse(message.createdAt);
}
