import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { ThemeProvider } from "../theme/ThemeProvider";
import { Button } from "./Button";

jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn().mockResolvedValue(null),
  setItemAsync: jest.fn(),
  deleteItemAsync: jest.fn()
}));

describe("Button", () => {
  it("fires onPress and exposes an accessible label", () => {
    const onPress = jest.fn();
    render(
      <ThemeProvider>
        <Button label="J'aime" onPress={onPress} />
      </ThemeProvider>
    );
    fireEvent.press(screen.getByRole("button", { name: "J'aime" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("does not fire while loading", () => {
    const onPress = jest.fn();
    render(
      <ThemeProvider>
        <Button label="Envoyer" onPress={onPress} loading />
      </ThemeProvider>
    );
    fireEvent.press(screen.getByRole("button", { name: "Envoyer" }));
    expect(onPress).not.toHaveBeenCalled();
  });
});
