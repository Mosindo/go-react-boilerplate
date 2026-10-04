import type { InfiniteData } from "@tanstack/react-query";

/** Cursor pages: `{ [key]: T[], nextCursor }`. These helpers are pure and generic over the list key. */
export type CursorPage<K extends string, T> = { nextCursor: string | null } & { [P in K]: T[] };
export type CursorData<K extends string, T> = InfiniteData<CursorPage<K, T>, unknown>;

export function emptyCursorData<K extends string, T>(): CursorData<K, T> {
  return { pages: [], pageParams: [] };
}

/** Flattens all pages into one list, dropping duplicates (first occurrence wins). */
export function flattenPages<K extends string, T extends { id: string }>(
  data: CursorData<K, T> | undefined,
  key: K
): T[] {
  if (!data) return [];
  const seen = new Set<string>();
  const out: T[] = [];
  for (const page of data.pages) {
    for (const item of page[key]) {
      if (!seen.has(item.id)) {
        seen.add(item.id);
        out.push(item);
      }
    }
  }
  return out;
}

/** Applies `fn` to the list of every page. */
export function mapPages<K extends string, T>(
  data: CursorData<K, T>,
  key: K,
  fn: (items: T[]) => T[]
): CursorData<K, T> {
  return { ...data, pages: data.pages.map((page) => ({ ...page, [key]: fn(page[key]) })) };
}

/** Puts an item at the very top (first page), removing any older copy from every page. */
export function prependItem<K extends string, T extends { id: string }>(
  data: CursorData<K, T> | undefined,
  key: K,
  item: T
): CursorData<K, T> {
  if (!data || data.pages.length === 0) {
    return { pages: [{ [key]: [item], nextCursor: null } as CursorPage<K, T>], pageParams: [null] };
  }
  const cleaned = mapPages(data, key, (items) => items.filter((existing) => existing.id !== item.id));
  const [first, ...rest] = cleaned.pages;
  return { ...cleaned, pages: [{ ...first, [key]: [item, ...first[key]] }, ...rest] };
}

/** Replaces an item (by id) wherever it is. Returns the same reference when absent. */
export function replaceItem<K extends string, T extends { id: string }>(
  data: CursorData<K, T> | undefined,
  key: K,
  id: string,
  update: (item: T) => T
): CursorData<K, T> | undefined {
  if (!data) return data;
  let found = false;
  const next = mapPages(data, key, (items) =>
    items.map((item) => {
      if (item.id !== id) return item;
      found = true;
      return update(item);
    })
  );
  return found ? next : data;
}

export function removeItems<K extends string, T>(
  data: CursorData<K, T> | undefined,
  key: K,
  predicate: (item: T) => boolean
): CursorData<K, T> | undefined {
  if (!data) return data;
  return mapPages(data, key, (items) => items.filter((item) => !predicate(item)));
}

export function hasItem<K extends string, T extends { id: string }>(
  data: CursorData<K, T> | undefined,
  key: K,
  id: string
): boolean {
  return !!data && data.pages.some((page) => page[key].some((item) => item.id === id));
}

/** `getNextPageParam` for useInfiniteQuery: undefined stops pagination. */
export function nextCursorOf(page: { nextCursor: string | null }): string | undefined {
  return page.nextCursor ?? undefined;
}
