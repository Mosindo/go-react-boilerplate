import { useMemo } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { darkColors, lightColors, type Palette } from "./tokens";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
};

const LIGHT: Theme = { scheme: "light", colors: lightColors };
const DARK: Theme = { scheme: "dark", colors: darkColors };

/** Follows the system appearance. */
export function useTheme(): Theme {
  return useColorScheme() === "dark" ? DARK : LIGHT;
}

/**
 * Builds a StyleSheet from the active palette. Pass a factory defined at
 * module level so the sheet is only rebuilt when the appearance changes.
 */
export function useStyles<T extends StyleSheet.NamedStyles<T>>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  return useMemo(() => StyleSheet.create(factory(colors)), [colors, factory]);
}
