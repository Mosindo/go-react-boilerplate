import { act, fireEvent, render, screen } from "@testing-library/react-native";
import React from "react";
import type { PublicProfile } from "../../../api/types";
import { SwipeDeck } from "../SwipeDeck";

const profile = (userId: string, name: string): PublicProfile => ({
  userId,
  displayName: name,
  age: 29,
  gender: "woman",
  bio: "Loves hiking",
  city: "Porto",
  interests: [{ slug: "hiking", label: "Hiking" }],
  photos: [],
  distanceKm: 10,
  commonInterests: 1
});

describe("SwipeDeck", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("shows the top card details", () => {
    render(
      <SwipeDeck
        cards={[profile("1", "Ana"), { ...profile("2", "Bea"), bio: "Likes jazz", interests: [] }]}
        onOpenProfile={jest.fn()}
        onSwipe={jest.fn()}
      />
    );
    expect(screen.getByText("Ana, 29")).toBeTruthy();
    expect(screen.getByText("Loves hiking")).toBeTruthy();
    expect(screen.getByText("Hiking")).toBeTruthy();
  });

  it("Like and Pass buttons report the right action for the top card", () => {
    const onSwipe = jest.fn();
    const cards = [profile("1", "Ana"), profile("2", "Bea")];
    render(<SwipeDeck cards={cards} onOpenProfile={jest.fn()} onSwipe={onSwipe} />);

    fireEvent.press(screen.getByRole("button", { name: "Like" }));
    act(() => {
      jest.advanceTimersByTime(500);
    });
    expect(onSwipe).toHaveBeenCalledWith(cards[0], "like");
  });

  it("Pass reports pass; double taps do not swipe twice while flinging", () => {
    const onSwipe = jest.fn();
    const cards = [profile("1", "Ana")];
    render(<SwipeDeck cards={cards} onOpenProfile={jest.fn()} onSwipe={onSwipe} />);
    fireEvent.press(screen.getByRole("button", { name: "Pass" }));
    fireEvent.press(screen.getByRole("button", { name: "Pass" }));
    act(() => {
      jest.advanceTimersByTime(500);
    });
    expect(onSwipe).toHaveBeenCalledTimes(1);
    expect(onSwipe).toHaveBeenCalledWith(cards[0], "pass");
  });

  it("opens the full profile from the card info", () => {
    const onOpenProfile = jest.fn();
    const cards = [profile("1", "Ana")];
    render(<SwipeDeck cards={cards} onOpenProfile={onOpenProfile} onSwipe={jest.fn()} />);
    fireEvent.press(screen.getByLabelText(/Ana, 29.*View full profile/));
    expect(onOpenProfile).toHaveBeenCalledWith(cards[0]);
  });

  it("disables the action buttons when the deck is empty", () => {
    render(<SwipeDeck cards={[]} onOpenProfile={jest.fn()} onSwipe={jest.fn()} />);
    expect(screen.getByRole("button", { name: "Like" }).props.accessibilityState.disabled).toBe(true);
  });
});
