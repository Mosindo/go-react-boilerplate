import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkPalette, lightPalette, radii, spacing, type, type Palette } from "./palette";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
  spacing: typeof spacing;
  radii: typeof radii;
  type: typeof type;
};

function buildTheme(scheme: "light" | "dark"): Theme {
  return { scheme, colors: scheme === "dark" ? darkPalette : lightPalette, spacing, radii, type };
}

const ThemeContext = createContext<Theme>(buildTheme("light"));

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const theme = useMemo(() => buildTheme(scheme), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoizes a style factory against the current theme. */
export function useThemedStyles<T>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => factory(theme), [factory, theme]);
}
