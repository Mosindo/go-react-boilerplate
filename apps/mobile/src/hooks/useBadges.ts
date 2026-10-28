import { totalUnread } from "../realtime/cache";
import { useConversations, useNotifications } from "./queries";

export function useUnreadMessages(): number {
  const { data } = useConversations();
  return totalUnread(data);
}

export function useUnreadNotifications(): number {
  const { data } = useNotifications();
  return data?.pages[0]?.unreadCount ?? 0;
}
