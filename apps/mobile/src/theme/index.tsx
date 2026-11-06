import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  primaryText: string;
  primarySoft: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
};

// Warm "ember" identity: cream and terracotta by day, aubergine and coral by night.
export const lightPalette: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  text: "#2A1F1D",
  textMuted: "#6F625D",
  border: "#E6DAD0",
  primary: "#C8442B",
  primaryText: "#FFFFFF",
  primarySoft: "#FBE4DD",
  success: "#2E7D5B",
  danger: "#B3261E",
  dangerSoft: "#FCE8E6",
  overlay: "rgba(42, 31, 29, 0.5)",
  scrim: "rgba(20, 10, 8, 0.62)"
};

export const darkPalette: Palette = {
  background: "#170F14",
  surface: "#231920",
  surfaceAlt: "#2E2229",
  text: "#F6EDE8",
  textMuted: "#B5A59F",
  border: "#3A2D34",
  primary: "#EE7A61",
  primaryText: "#170F14",
  primarySoft: "#3D2220",
  success: "#5CC49A",
  danger: "#F2867E",
  dangerSoft: "#3B1F1E",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(10, 5, 8, 0.7)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;

type Theme = { colors: Palette; isDark: boolean };

const ThemeContext = createContext<Theme>({ colors: lightPalette, isDark: false });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => ({ colors: scheme === "dark" ? darkPalette : lightPalette, isDark: scheme === "dark" }),
    [scheme]
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
