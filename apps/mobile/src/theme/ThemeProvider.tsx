import React, { createContext, useContext, useMemo } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./palette";

export type Theme = {
  dark: boolean;
  colors: Palette;
};

const ThemeContext = createContext<Theme>({ dark: false, colors: lightPalette });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => (scheme === "dark" ? { dark: true, colors: darkPalette } : { dark: false, colors: lightPalette }),
    [scheme]
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/**
 * Builds theme-aware styles once per palette:
 *   const useStyles = makeStyles((c) => ({ box: { backgroundColor: c.surface } }));
 */
export function makeStyles<T extends StyleSheet.NamedStyles<T>>(factory: (colors: Palette, dark: boolean) => T) {
  const cache = new WeakMap<Palette, T>();
  return function useStyles(): T {
    const { colors, dark } = useTheme();
    let styles = cache.get(colors);
    if (!styles) {
      styles = StyleSheet.create(factory(colors, dark));
      cache.set(colors, styles);
    }
    return styles;
  };
}
