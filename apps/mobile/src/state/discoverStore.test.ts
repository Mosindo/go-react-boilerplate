import { act, renderHook } from "@testing-library/react-native";
import {
  bumpDiscoverVersion,
  markProfileResolved,
  resetDiscoverStore,
  useDiscoverStore
} from "./discoverStore";

describe("discover store", () => {
  beforeEach(() => resetDiscoverStore());

  it("tracks resolved profiles and resets them when the version changes", () => {
    const { result } = renderHook(() => useDiscoverStore());
    expect(result.current.resolved.size).toBe(0);

    act(() => markProfileResolved("a"));
    expect(result.current.resolved.has("a")).toBe(true);

    const before = result.current.version;
    act(() => bumpDiscoverVersion());
    expect(result.current.version).toBe(before + 1);
    expect(result.current.resolved.size).toBe(0);
  });
});
