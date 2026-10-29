import { act, renderHook, waitFor } from "@testing-library/react-native";
import * as discovery from "../api/discovery";
import { ApiError } from "../api/client";
import type { PublicProfile } from "../api/types";
import { resetDiscoverStore } from "../state/discoverStore";
import { useDeck } from "./useDeck";

jest.mock("../api/discovery");
const api = discovery as jest.Mocked<typeof discovery>;

const profile = (id: string): PublicProfile => ({
  id,
  firstName: id.toUpperCase(),
  age: 30,
  gender: "woman",
  bio: "",
  city: "Paris",
  distanceKm: 3,
  interests: [],
  photos: []
});

describe("useDeck", () => {
  beforeEach(() => {
    resetDiscoverStore();
    jest.resetAllMocks();
  });

  it("loads a batch, skips duplicates on refill and removes swiped profiles", async () => {
    api.fetchDiscover
      .mockResolvedValueOnce([profile("a"), profile("b")])
      .mockResolvedValue([profile("b"), profile("c")]);
    api.swipe.mockResolvedValue({
      action: "like",
      alreadyProcessed: false,
      matched: false,
      match: null
    });

    const { result } = renderHook(() => useDeck(jest.fn(), jest.fn()));
    await waitFor(() => expect(result.current.deck.map((p) => p.id)).toEqual(["a", "b", "c"]));

    await act(async () => {
      await result.current.act(result.current.deck[0] as PublicProfile, "like");
    });
    expect(api.swipe).toHaveBeenCalledWith("a", "like");
    expect(result.current.deck.map((p) => p.id)).toEqual(["b", "c"]);
  });

  it("announces a fresh match", async () => {
    const match = {
      id: "m1",
      conversationId: "c1",
      user: { id: "a", firstName: "A", photoUrl: null }
    };
    api.fetchDiscover.mockResolvedValue([profile("a")]);
    api.swipe.mockResolvedValue({ action: "like", alreadyProcessed: false, matched: true, match });
    const onMatch = jest.fn();

    const { result } = renderHook(() => useDeck(onMatch, jest.fn()));
    await waitFor(() => expect(result.current.deck).toHaveLength(1));
    await act(async () => {
      await result.current.act(result.current.deck[0] as PublicProfile, "like");
    });
    expect(onMatch).toHaveBeenCalledWith(match);
  });

  it("puts the profile back and reports a network failure", async () => {
    api.fetchDiscover.mockResolvedValueOnce([profile("a")]).mockResolvedValue([]);
    api.swipe.mockRejectedValue(new ApiError(0, "Connexion impossible."));
    const onProblem = jest.fn();

    const { result } = renderHook(() => useDeck(jest.fn(), onProblem));
    await waitFor(() => expect(result.current.deck).toHaveLength(1));
    await act(async () => {
      await result.current.act(result.current.deck[0] as PublicProfile, "pass");
    });
    expect(result.current.deck.map((p) => p.id)).toEqual(["a"]);
    expect(onProblem).toHaveBeenCalledWith("Connexion impossible.");
  });

  it("drops a profile the server no longer offers", async () => {
    api.fetchDiscover.mockResolvedValueOnce([profile("a")]).mockResolvedValue([]);
    api.swipe.mockRejectedValue(new ApiError(404, "profile not available"));
    const onProblem = jest.fn();

    const { result } = renderHook(() => useDeck(jest.fn(), onProblem));
    await waitFor(() => expect(result.current.deck).toHaveLength(1));
    await act(async () => {
      await result.current.act(result.current.deck[0] as PublicProfile, "like");
    });
    expect(result.current.deck).toHaveLength(0);
    expect(onProblem).not.toHaveBeenCalled();
  });
});
