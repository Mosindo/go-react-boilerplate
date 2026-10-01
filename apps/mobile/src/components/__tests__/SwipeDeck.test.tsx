import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react-native";
import type { PublicProfile } from "../../api/types";
import { ThemeProvider } from "../../shared/ui";
import { SwipeDeck } from "../SwipeDeck";

jest.mock("../../hooks/useAuth", () => ({ useAuth: () => ({ accessToken: "token" }) }));

const profile = (id: string, name: string): PublicProfile => ({
  id,
  firstName: name,
  age: 29,
  gender: "woman",
  bio: "",
  city: "Lyon",
  distanceKm: 4,
  interests: [{ slug: "music", label: "Musique" }],
  photos: [{ id: `p-${id}`, position: 0, url: `/photos/p-${id}/file` }]
});

function setup(onSwipe = jest.fn(), onOpenProfile = jest.fn()) {
  const profiles = [profile("a", "Camille"), profile("b", "Léa")];
  render(
    <ThemeProvider forceScheme="light">
      <SwipeDeck onOpenProfile={onOpenProfile} onSwipe={onSwipe} profiles={profiles} />
    </ThemeProvider>
  );
  return { onSwipe, onOpenProfile, profiles };
}

describe("SwipeDeck", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("shows the top card with name, age and approximate distance", () => {
    setup();
    expect(screen.getByText("Camille, 29")).toBeTruthy();
    expect(screen.getAllByText("Lyon · À 4 km").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Musique").length).toBeGreaterThan(0);
  });

  it("likes the top card with the button (swipe alternative for accessibility)", () => {
    const { onSwipe, profiles } = setup();
    fireEvent.press(screen.getByRole("button", { name: "J'aime" }));
    act(() => {
      jest.advanceTimersByTime(400);
    });
    expect(onSwipe).toHaveBeenCalledTimes(1);
    expect(onSwipe).toHaveBeenCalledWith(profiles[0], "like");
  });

  it("passes with the button and ignores double taps while animating", () => {
    const { onSwipe, profiles } = setup();
    const pass = screen.getByRole("button", { name: "Passer" });
    fireEvent.press(pass);
    fireEvent.press(pass);
    act(() => {
      jest.advanceTimersByTime(400);
    });
    expect(onSwipe).toHaveBeenCalledTimes(1);
    expect(onSwipe).toHaveBeenCalledWith(profiles[0], "pass");
  });

  it("opens the full profile", () => {
    const { onOpenProfile, profiles } = setup();
    fireEvent.press(screen.getByRole("button", { name: "Voir le profil complet" }));
    expect(onOpenProfile).toHaveBeenCalledWith(profiles[0]);
  });

  it("disables actions when the deck is empty", () => {
    render(
      <ThemeProvider forceScheme="light">
        <SwipeDeck onOpenProfile={jest.fn()} onSwipe={jest.fn()} profiles={[]} />
      </ThemeProvider>
    );
    expect(screen.getByRole("button", { name: "J'aime" }).props.accessibilityState).toMatchObject({ disabled: true });
  });
});
