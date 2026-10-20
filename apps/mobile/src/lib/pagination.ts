export type Page<T> = { items: T[]; nextCursor: string | null };

export function dedupeById<T extends { id: string }>(items: T[]): T[] {
  const seen = new Set<string>();
  const result: T[] = [];
  for (const item of items) {
    if (seen.has(item.id)) continue;
    seen.add(item.id);
    result.push(item);
  }
  return result;
}

export function flattenPages<T extends { id: string }>(pages: Page<T>[] | undefined): T[] {
  if (!pages) return [];
  return dedupeById(pages.flatMap((page) => page.items));
}
