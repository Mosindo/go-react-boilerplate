import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { ThemeProvider } from "../theme";
import { Button, Chip, EmptyState } from "../ui";

jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));

const wrap = (node: React.ReactElement) => render(<ThemeProvider>{node}</ThemeProvider>);

describe("ui kit", () => {
  it("Button fires and exposes accessibility state", () => {
    const onPress = jest.fn();
    wrap(<Button label="Continuer" onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("disabled Button does not fire", () => {
    const onPress = jest.fn();
    wrap(<Button label="Envoyer" onPress={onPress} disabled />);
    fireEvent.press(screen.getByRole("button", { name: "Envoyer" }));
    expect(onPress).not.toHaveBeenCalled();
  });

  it("Chip reports its checked state", () => {
    wrap(<Chip label="Randonnée" selected onPress={() => undefined} />);
    expect(screen.getByRole("checkbox", { name: "Randonnée" }).props.accessibilityState).toMatchObject({ checked: true });
  });

  it("EmptyState renders its action", () => {
    const onAction = jest.fn();
    wrap(<EmptyState icon="compass-outline" title="Vous avez fait le tour" actionLabel="Élargir" onAction={onAction} />);
    expect(screen.getByText("Vous avez fait le tour")).toBeTruthy();
    fireEvent.press(screen.getByRole("button", { name: "Élargir" }));
    expect(onAction).toHaveBeenCalled();
  });
});
