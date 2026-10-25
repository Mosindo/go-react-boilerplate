import { infiniteQueryOptions, type InfiniteData, type QueryKey } from "@tanstack/react-query";
import type { Page } from "../api";

/** Standard options for a cursor-paginated endpoint (`{ items, nextCursor }`). */
export function cursorQuery<T, TKey extends QueryKey>(
  queryKey: TKey,
  fetchPage: (cursor: string | undefined) => Promise<Page<T>>
) {
  return infiniteQueryOptions({
    queryKey,
    queryFn: ({ pageParam }) => fetchPage(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined
  });
}

export function flattenPages<T>(data: InfiniteData<{ items: T[] }> | undefined): T[] {
  return data ? data.pages.flatMap((page) => page.items) : [];
}
