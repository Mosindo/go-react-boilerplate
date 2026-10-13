import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react-native";
import { fetchDiscover, swipe } from "../api/platform";
import type { PublicProfile } from "../api/types";
import { renderWithProviders } from "../test-utils";
import DiscoverScreen from "./DiscoverScreen";

const mockNavigate = jest.fn();

jest.mock("@react-navigation/native", () => ({ useNavigation: () => ({ navigate: mockNavigate }) }));
jest.mock("../api/platform", () => ({ fetchDiscover: jest.fn(), swipe: jest.fn() }));
jest.mock("../hooks/queries", () => ({ useOwnProfile: () => ({ data: { photos: [{ id: "mine", position: 0 }] } }) }));
jest.mock("../features/safety/SafetyMenu", () => ({ SafetyMenu: () => null }));

const person = (id: string, firstName: string): PublicProfile => ({
  id,
  firstName,
  age: 29,
  gender: "man",
  bio: "Bonjour",
  city: "Paris",
  distanceKm: 10,
  interests: [{ id: 1, slug: "travel", label: "Voyages" }],
  photos: [{ id: `photo-${id}`, position: 0 }]
});

describe("DiscoverScreen", () => {
  beforeEach(() => jest.clearAllMocks());

  it("shows the first profile with its essentials", async () => {
    jest
      .mocked(fetchDiscover)
      .mockResolvedValueOnce([person("1", "Bob"), person("2", "Leo")])
      .mockResolvedValue([]);
    renderWithProviders(<DiscoverScreen />);
    expect(await screen.findByText("Bob, 29")).toBeTruthy();
    expect(screen.getAllByText("À 10 km · Paris").length).toBeGreaterThan(0);
  });

  it("likes the top profile and announces a match", async () => {
    jest
      .mocked(fetchDiscover)
      .mockResolvedValueOnce([person("1", "Bob")])
      .mockResolvedValue([]);
    jest.mocked(swipe).mockResolvedValue({ matched: true, match: { id: "m1", conversationId: "c1" } });
    renderWithProviders(<DiscoverScreen />);
    await screen.findByText("Bob, 29");

    fireEvent.press(screen.getByTestId("swipe-like"));
    await waitFor(() => expect(swipe).toHaveBeenCalledWith("1", "like"));
    expect(await screen.findByTestId("match-modal")).toBeTruthy();

    fireEvent.press(screen.getByTestId("match-send-message"));
    expect(mockNavigate).toHaveBeenCalledWith("Conversation", expect.objectContaining({ conversationId: "c1", userId: "1", name: "Bob" }));
  });

  it("passes on a profile and ends on the empty state", async () => {
    jest
      .mocked(fetchDiscover)
      .mockResolvedValueOnce([person("1", "Bob")])
      .mockResolvedValue([]);
    jest.mocked(swipe).mockResolvedValue({ matched: false });
    renderWithProviders(<DiscoverScreen />);
    await screen.findByText("Bob, 29");

    fireEvent.press(screen.getByTestId("swipe-pass"));
    await waitFor(() => expect(swipe).toHaveBeenCalledWith("1", "pass"));
    expect(await screen.findByText("Vous avez tout vu !")).toBeTruthy();
  });
});
