import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkPalette, lightPalette, radii, spacing, typography, type Palette } from "./palette";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
  spacing: typeof spacing;
  radii: typeof radii;
  typography: typeof typography;
};

const ThemeContext = createContext<Theme | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const systemScheme = useColorScheme();
  const scheme = systemScheme === "light" ? "light" : "dark";
  const value = useMemo<Theme>(() => ({ scheme, colors: scheme === "light" ? lightPalette : darkPalette, spacing, radii, typography }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  const theme = useContext(ThemeContext);
  if (!theme) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return theme;
}
