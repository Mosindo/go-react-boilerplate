import { fireEvent, render, screen } from "@testing-library/react-native";
import React from "react";
import { PreferencesFields, DEFAULT_PREFERENCES } from "../components/PreferencesForm";
import { ThemeProvider } from "../theme";
import { Button, Chip } from "../ui";

jest.mock("expo-secure-store", () => ({ getItemAsync: jest.fn(async () => null), setItemAsync: jest.fn(async () => undefined) }));

jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));

const wrap = (node: React.ReactElement) => render(<ThemeProvider>{node}</ThemeProvider>);

describe("ui kit", () => {
  it("Button fires onPress and is inert while loading", () => {
    const onPress = jest.fn();
    const { rerender } = wrap(<Button label="Continuer" onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).toHaveBeenCalledTimes(1);

    rerender(
      <ThemeProvider>
        <Button label="Continuer" onPress={onPress} loading />
      </ThemeProvider>
    );
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("Chip exposes its selected state", () => {
    wrap(<Chip label="Voyage" selected onPress={() => undefined} />);
    expect(screen.getByRole("button", { name: "Voyage" })).toBeSelected();
  });
});

describe("PreferencesFields", () => {
  it("toggles genders and keeps the age range consistent", () => {
    const onChange = jest.fn();
    wrap(<PreferencesFields value={{ ...DEFAULT_PREFERENCES, ageMin: 30, ageMax: 30 }} onChange={onChange} />);

    fireEvent.press(screen.getByRole("button", { name: "Femme" }));
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ interestedIn: ["woman"] }));

    // min cannot exceed max: the + on "Âge min." is disabled when equal
    fireEvent.press(screen.getByRole("button", { name: "Augmenter Âge min." }));
    expect(onChange).toHaveBeenCalledTimes(1);

    fireEvent.press(screen.getByRole("button", { name: "Augmenter Âge max." }));
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ ageMax: 31 }));
  });
});
