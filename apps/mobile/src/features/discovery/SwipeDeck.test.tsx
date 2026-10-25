import React from "react";
import { Animated } from "react-native";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { publicProfile } from "../../test-utils";
import { SwipeDeck } from "./SwipeDeck";

jest.mock("../../shared/ui/AuthImage", () => ({ AuthImage: () => null }));

describe("SwipeDeck buttons", () => {
  beforeEach(() => {
    jest.spyOn(Animated, "timing").mockImplementation(() => ({
      start: (callback?: Animated.EndCallback) => callback?.({ finished: true }),
      stop: jest.fn(),
      reset: jest.fn()
    }));
  });
  afterEach(() => jest.restoreAllMocks());

  const queue = [publicProfile({ userId: "a", firstName: "Ana" }), publicProfile({ userId: "b", firstName: "Bob" })];

  it("likes the top card with the explicit button", async () => {
    const onSwipe = jest.fn();
    await render(<SwipeDeck onOpenProfile={jest.fn()} onSwipe={onSwipe} queue={queue} />);
    await fireEvent.press(screen.getByTestId("deck-like-button"));
    expect(onSwipe).toHaveBeenCalledWith(queue[0], "like");
  });

  it("passes the top card with the explicit button", async () => {
    const onSwipe = jest.fn();
    await render(<SwipeDeck onOpenProfile={jest.fn()} onSwipe={onSwipe} queue={queue} />);
    await fireEvent.press(screen.getByTestId("deck-pass-button"));
    expect(onSwipe).toHaveBeenCalledWith(queue[0], "pass");
  });

  it("opens the full profile from the info button", async () => {
    const onOpenProfile = jest.fn();
    await render(<SwipeDeck onOpenProfile={onOpenProfile} onSwipe={jest.fn()} queue={queue} />);
    await fireEvent.press(screen.getByTestId("deck-open-profile"));
    expect(onOpenProfile).toHaveBeenCalledWith(queue[0]);
  });

  it("renders nothing for an empty queue", async () => {
    await render(<SwipeDeck onOpenProfile={jest.fn()} onSwipe={jest.fn()} queue={[]} />);
    expect(screen.queryByTestId("deck-like-button")).toBeNull();
  });
});
