import { useColorScheme } from "react-native";
import { paletteFor, type ColorPalette, type ColorScheme } from "./palette";

export type Theme = {
  scheme: ColorScheme;
  isDark: boolean;
  colors: ColorPalette;
  /** Value for expo-status-bar's `style`. */
  statusBarStyle: "light" | "dark";
};

const themes: Record<ColorScheme, Theme> = {
  light: { scheme: "light", isDark: false, colors: paletteFor("light"), statusBarStyle: "dark" },
  dark: { scheme: "dark", isDark: true, colors: paletteFor("dark"), statusBarStyle: "light" }
};

/** Follows the system colour scheme. The returned object is referentially stable per scheme. */
export function useTheme(): Theme {
  return themes[useColorScheme() === "dark" ? "dark" : "light"];
}
