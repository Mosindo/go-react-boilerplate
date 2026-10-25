import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi, matchesApi, queryKeys } from "../../api";
import { cursorQuery } from "../../shared/pagination";

const matchesQuery = cursorQuery(queryKeys.matches, matchesApi.list);
const conversationsQuery = cursorQuery(queryKeys.conversations, chatApi.conversations);

export function useMatches() {
  return useInfiniteQuery(matchesQuery);
}

export function useConversations() {
  return useInfiniteQuery(conversationsQuery);
}
