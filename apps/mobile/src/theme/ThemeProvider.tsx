import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./palette";

const ThemeContext = createContext<Palette>(lightPalette);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const palette = useMemo(() => (scheme === "dark" ? darkPalette : lightPalette), [scheme]);
  return <ThemeContext.Provider value={palette}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Palette {
  return useContext(ThemeContext);
}
