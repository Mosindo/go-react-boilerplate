import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react-native";
import type { Profile } from "../api/types";
import { SwipeDeck } from "../features/discover/SwipeDeck";

jest.mock("../ui/RemoteImage", () => ({ RemoteImage: () => null }));

const profile = (id: string, name: string): Profile => ({
  userId: id,
  firstName: name,
  age: 29,
  gender: "woman",
  bio: "",
  city: "Lyon",
  interests: [{ slug: "hiking", label: "Hiking" }],
  photos: [{ id: `${id}-p`, position: 0, url: `/photos/${id}-p/file` }],
  distanceKm: 5
});

describe("SwipeDeck", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("likes the top profile with the accessible button", () => {
    const onSwipe = jest.fn();
    render(
      <SwipeDeck profiles={[profile("1", "Ana"), profile("2", "Bea")]} onSwipe={onSwipe} onOpen={jest.fn()} />
    );
    fireEvent.press(screen.getByTestId("like-button"));
    act(() => {
      jest.advanceTimersByTime(500);
    });
    expect(onSwipe).toHaveBeenCalledTimes(1);
    expect(onSwipe.mock.calls[0][0].userId).toBe("1");
    expect(onSwipe.mock.calls[0][1]).toBe("like");
  });

  it("passes and opens the full profile", () => {
    const onSwipe = jest.fn();
    const onOpen = jest.fn();
    render(<SwipeDeck profiles={[profile("1", "Ana")]} onSwipe={onSwipe} onOpen={onOpen} />);
    fireEvent.press(screen.getByTestId("open-profile"));
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ userId: "1" }));
    fireEvent.press(screen.getByTestId("pass-button"));
    act(() => {
      jest.advanceTimersByTime(500);
    });
    expect(onSwipe).toHaveBeenCalledWith(expect.objectContaining({ userId: "1" }), "pass");
  });

  it("renders nothing without profiles", () => {
    const { toJSON } = render(<SwipeDeck profiles={[]} onSwipe={jest.fn()} onOpen={jest.fn()} />);
    expect(toJSON()).toBeNull();
  });
});
