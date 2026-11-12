import React, { createContext, useContext, useMemo } from "react";
import { Platform, StyleSheet, useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./palette";

export type Theme = {
  colors: Palette;
  isDark: boolean;
  serif: string | undefined;
};

const ThemeContext = createContext<Theme | null>(null);

// A serif for the wordmark and display titles gives the brand its editorial warmth.
const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const theme = useMemo<Theme>(() => ({ colors: scheme === "dark" ? darkPalette : lightPalette, isDark: scheme === "dark", serif }), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  const theme = useContext(ThemeContext);
  if (!theme) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return theme;
}

/** Builds a memoized StyleSheet from the current theme. */
export function useStyles<T extends StyleSheet.NamedStyles<T>>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => StyleSheet.create(factory(theme)), [factory, theme]);
}
