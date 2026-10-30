import { describe, expect, it } from "vitest";
import type { PublicProfile } from "../api/types";
import { mergeBatch } from "./queue";

const profile = (id: string): PublicProfile => ({
  userId: id,
  displayName: id,
  age: 30,
  gender: "woman",
  bio: "",
  city: "",
  interests: [],
  photos: []
});

describe("mergeBatch", () => {
  it("never queues a profile twice or one that was already decided", () => {
    const queue = [profile("a")];
    const merged = mergeBatch(queue, [profile("a"), profile("b"), profile("c")], new Set(["c"]));
    expect(merged.map((p) => p.userId)).toEqual(["a", "b"]);
  });
});
