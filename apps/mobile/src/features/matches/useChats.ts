import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";

const PAGE = 20;

export function useChats(enabled = true) {
  return useInfiniteQuery({
    queryKey: queryKeys.chats,
    enabled,
    initialPageParam: 0,
    queryFn: async ({ pageParam }) => (await chatApi.list(pageParam, PAGE)).chats,
    getNextPageParam: (lastPage, allPages) => (lastPage.length === PAGE ? allPages.length * PAGE : undefined),
    staleTime: 15_000
  });
}
