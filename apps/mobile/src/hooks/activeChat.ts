import { createContext, useContext, type MutableRefObject } from "react";

/** The member whose conversation is on screen (suppresses redundant toasts, focuses resync). */
export const ActiveChatContext = createContext<MutableRefObject<string | null>>({ current: null });

export function useActiveChat() {
  return useContext(ActiveChatContext);
}
