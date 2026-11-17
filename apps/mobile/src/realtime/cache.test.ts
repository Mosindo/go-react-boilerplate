import { test } from "node:test";
import assert from "node:assert/strict";
import { insertMessage } from "./cache.ts";

const base = { conversationId: "c", senderId: "u", content: "x", createdAt: "2026-01-01T00:00:00Z" };

test("insertMessage replaces the optimistic copy and ignores duplicates", () => {
  const data = {
    pageParams: [null],
    pages: [{ messages: [{ ...base, id: "local-1", clientMessageId: "k1" }], nextCursor: null, otherLastReadAt: null }]
  };
  const server = { ...base, id: "m1", clientMessageId: "k1" };
  const once = insertMessage(data, server);
  assert.deepEqual(
    once?.pages[0].messages.map((m) => m.id),
    ["m1"]
  );
  const twice = insertMessage(once, server);
  assert.deepEqual(
    twice?.pages[0].messages.map((m) => m.id),
    ["m1"]
  );
  const other = insertMessage(twice, { ...base, id: "m2" });
  assert.deepEqual(
    other?.pages[0].messages.map((m) => m.id),
    ["m2", "m1"]
  );
  assert.equal(insertMessage(undefined, server), undefined);
});
