import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme, type ColorSchemeName } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./palette";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
};

const ThemeContext = createContext<Theme>({ scheme: "light", colors: lightPalette });

export function resolveTheme(scheme: ColorSchemeName): Theme {
  return scheme === "dark" ? { scheme: "dark", colors: darkPalette } : { scheme: "light", colors: lightPalette };
}

/** Follows the system appearance (light/dark). */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const theme = useMemo(() => resolveTheme(scheme), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised style factory bound to the current palette. */
export function useThemedStyles<T>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  return useMemo(() => factory(colors), [colors, factory]);
}
