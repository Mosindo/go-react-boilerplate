import { act, renderHook, waitFor } from "@testing-library/react-native";
import { ApiError } from "../../api/client";
import { fetchDiscover, swipe } from "../../api/platform";
import type { PublicProfile } from "../../api/types";
import { useDeck } from "./useDeck";

jest.mock("../../api/platform");

const mockedFetch = jest.mocked(fetchDiscover);
const mockedSwipe = jest.mocked(swipe);

function profile(id: string): PublicProfile {
  return {
    id,
    firstName: id,
    age: 30,
    gender: "woman",
    bio: "",
    city: null,
    distanceKm: 10,
    interests: [],
    photos: [{ id: `ph-${id}`, position: 0 }]
  };
}

describe("useDeck", () => {
  beforeEach(() => jest.clearAllMocks());

  it("loads profiles and removes a card once swiped", async () => {
    mockedFetch.mockResolvedValueOnce([profile("a"), profile("b")]).mockResolvedValue([]);
    mockedSwipe.mockResolvedValue({ matched: false });
    const onMatch = jest.fn();
    const { result } = renderHook(() => useDeck({ onMatch }));

    await waitFor(() => expect(result.current.status).toBe("ready"));
    expect(result.current.queue.map((p) => p.id)).toEqual(["a", "b"]);

    await act(async () => {
      await result.current.act(result.current.queue[0], "like");
    });
    expect(mockedSwipe).toHaveBeenCalledWith("a", "like");
    expect(result.current.queue.map((p) => p.id)).toEqual(["b"]);
    expect(onMatch).not.toHaveBeenCalled();
  });

  it("announces a match", async () => {
    mockedFetch.mockResolvedValueOnce([profile("a")]).mockResolvedValue([]);
    mockedSwipe.mockResolvedValue({ matched: true, match: { id: "m1", conversationId: "c1" } });
    const onMatch = jest.fn();
    const { result } = renderHook(() => useDeck({ onMatch }));
    await waitFor(() => expect(result.current.status).toBe("ready"));

    await act(async () => {
      await result.current.act(result.current.queue[0], "like");
    });
    expect(onMatch).toHaveBeenCalledWith(expect.objectContaining({ id: "a" }), {
      matched: true,
      match: { id: "m1", conversationId: "c1" }
    });
  });

  it("never shows a profile twice, even if the server repeats it", async () => {
    mockedFetch
      .mockResolvedValueOnce([profile("a"), profile("b")])
      .mockResolvedValueOnce([profile("b"), profile("c")])
      .mockResolvedValue([]);
    mockedSwipe.mockResolvedValue({ matched: false });
    const { result } = renderHook(() => useDeck({ onMatch: jest.fn() }));
    await waitFor(() => expect(result.current.queue.length).toBeGreaterThan(0));
    await waitFor(() => expect(result.current.queue.map((p) => p.id)).toEqual(["a", "b", "c"]));
  });

  it("restores the card when the swipe fails, but drops it on a conflict", async () => {
    mockedFetch.mockResolvedValueOnce([profile("a"), profile("b")]).mockResolvedValue([]);
    mockedSwipe.mockRejectedValueOnce(new ApiError(0, "offline")).mockRejectedValueOnce(new ApiError(409, "already processed"));
    const { result } = renderHook(() => useDeck({ onMatch: jest.fn() }));
    await waitFor(() => expect(result.current.status).toBe("ready"));

    await act(async () => {
      await result.current.act(result.current.queue[0], "pass");
    });
    expect(result.current.queue.map((p) => p.id)).toEqual(["a", "b"]);

    await act(async () => {
      await result.current.act(result.current.queue[0], "pass");
    });
    expect(result.current.queue.map((p) => p.id)).toEqual(["b"]);
  });

  it("reports an empty deck and an incomplete profile", async () => {
    mockedFetch.mockResolvedValueOnce([]);
    const empty = renderHook(() => useDeck({ onMatch: jest.fn() }));
    await waitFor(() => expect(empty.result.current.status).toBe("empty"));

    mockedFetch.mockRejectedValueOnce(new ApiError(409, "complete your profile"));
    const incomplete = renderHook(() => useDeck({ onMatch: jest.fn() }));
    await waitFor(() => expect(incomplete.result.current.status).toBe("incomplete"));
  });
});
