import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./tokens";

type Theme = {
  colors: Palette;
  isDark: boolean;
};

const ThemeContext = createContext<Theme>({ colors: darkPalette, isDark: true });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  // Dark is the signature look; light follows the OS preference.
  const isDark = scheme !== "light";
  const value = useMemo<Theme>(() => ({ colors: isDark ? darkPalette : lightPalette, isDark }), [isDark]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised StyleSheet factory bound to the current palette. */
export function useThemedStyles<T>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  return useMemo(() => factory(colors), [colors, factory]);
}
