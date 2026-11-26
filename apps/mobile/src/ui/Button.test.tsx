import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { Button } from "./Button";

describe("Button", () => {
  it("calls onPress and exposes an accessible label", () => {
    const onPress = jest.fn();
    render(<Button label="Liker" onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Liker" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("is disabled while loading", () => {
    const onPress = jest.fn();
    render(<Button label="Envoyer" loading onPress={onPress} />);
    const button = screen.getByRole("button", { name: "Envoyer" });
    expect(button).toBeDisabled();
    fireEvent.press(button);
    expect(onPress).not.toHaveBeenCalled();
  });
});
