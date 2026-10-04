import {
  flattenPages,
  hasItem,
  nextCursorOf,
  prependItem,
  removeItems,
  replaceItem,
  type CursorData
} from "../pagination";

type Item = { id: string; v?: number };
const data = (...lists: string[][]): CursorData<"items", Item> => ({
  pages: lists.map((ids, i) => ({
    items: ids.map((id) => ({ id })),
    nextCursor: i < lists.length - 1 ? `c${i}` : null
  })),
  pageParams: lists.map((_, i) => (i === 0 ? null : `c${i - 1}`))
});

describe("pagination helpers", () => {
  it("flattens pages in order and drops duplicates across pages", () => {
    expect(flattenPages(data(["a", "b"], ["b", "c"]), "items").map((i) => i.id)).toEqual(["a", "b", "c"]);
    expect(flattenPages(undefined, "items")).toEqual([]);
  });

  it("prepends to the first page and removes the older copy", () => {
    const next = prependItem(data(["a", "b"], ["c", "d"]), "items", { id: "d" });
    expect(next.pages[0].items.map((i) => i.id)).toEqual(["d", "a", "b"]);
    expect(next.pages[1].items.map((i) => i.id)).toEqual(["c"]);
  });

  it("creates a first page when there is none", () => {
    const next = prependItem<"items", Item>(undefined, "items", { id: "x" });
    expect(next.pages).toHaveLength(1);
    expect(next.pages[0].items).toEqual([{ id: "x" }]);
  });

  it("replaces, removes and finds items", () => {
    const d = data(["a"], ["b"]);
    expect(replaceItem(d, "items", "b", (i) => ({ ...i, v: 1 }))?.pages[1].items[0]).toEqual({ id: "b", v: 1 });
    expect(replaceItem(d, "items", "zzz", (i) => i)).toBe(d);
    expect(removeItems(d, "items", (i) => i.id === "a")?.pages[0].items).toEqual([]);
    expect(hasItem(d, "items", "b")).toBe(true);
    expect(hasItem(d, "items", "q")).toBe(false);
  });

  it("derives the next page param from the cursor", () => {
    expect(nextCursorOf({ nextCursor: "abc" })).toBe("abc");
    expect(nextCursorOf({ nextCursor: null })).toBeUndefined();
  });
});
