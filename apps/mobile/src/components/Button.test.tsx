import { fireEvent, render, screen } from "@testing-library/react-native";
import React from "react";
import { Button } from "./Button";

describe("Button", () => {
  it("calls onPress", () => {
    const onPress = jest.fn();
    render(<Button onPress={onPress} title="Continuer" />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("ignores presses while loading or disabled", () => {
    const onPress = jest.fn();
    const { rerender } = render(<Button loading onPress={onPress} title="Continuer" />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    rerender(<Button disabled onPress={onPress} title="Continuer" />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).not.toHaveBeenCalled();
  });
});
