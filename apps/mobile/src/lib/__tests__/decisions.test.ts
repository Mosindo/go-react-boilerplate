import { afterEach, describe, expect, it, vi } from "vitest";
import { clearDecisions, isDecided, markDecided, subscribeDecisions } from "../decisions";

afterEach(clearDecisions);

describe("decisions", () => {
  it("notifies subscribers once per new id", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeDecisions(listener);
    markDecided("a");
    markDecided("a");
    markDecided("b");
    expect(listener).toHaveBeenCalledTimes(2);
    expect(isDecided("a")).toBe(true);
    unsubscribe();
    markDecided("c");
    expect(listener).toHaveBeenCalledTimes(2);
  });

  it("forgets everything on clear", () => {
    markDecided("a");
    clearDecisions();
    expect(isDecided("a")).toBe(false);
  });
});
