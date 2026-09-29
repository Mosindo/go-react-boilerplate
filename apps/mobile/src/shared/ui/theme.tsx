import React, { createContext, useContext, useMemo, type ReactNode } from "react";
import { useColorScheme } from "react-native";
import { fontSizes, fontWeights, radii, shadows, spacing, typography } from "./tokens";

/**
 * Alba palette: warm cream / plum ink / terracotta. Light and dark variants follow the OS setting.
 * Components must read colours through `useTheme()` (never hard-code hex values) so dark mode works everywhere.
 */
export type Colors = {
  background: string;
  backgroundElevated: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primarySoft: string;
  primaryForeground: string;
  secondary: string;
  success: string;
  successSoft: string;
  warning: string;
  warningSoft: string;
  danger: string;
  dangerSoft: string;
  like: string;
  pass: string;
  overlay: string;
  photoScrim: string;
};

export const lightColors: Colors = {
  background: "#FBF7F2",
  backgroundElevated: "#FFFFFF",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  border: "#E8DED3",
  borderStrong: "#D2C4B6",
  text: "#2A1F2D",
  textMuted: "#6F6472",
  textSubtle: "#9A8F9C",
  primary: "#C2452D",
  primarySoft: "#FBE9E4",
  primaryForeground: "#FFFFFF",
  secondary: "#4A2545",
  success: "#2E7D5B",
  successSoft: "#E3F3EB",
  warning: "#A5620F",
  warningSoft: "#FBF0DC",
  danger: "#B3261E",
  dangerSoft: "#FCE8E6",
  like: "#C2452D",
  pass: "#6F6472",
  overlay: "rgba(42, 31, 45, 0.45)",
  photoScrim: "rgba(20, 10, 22, 0.55)"
};

export const darkColors: Colors = {
  background: "#171117",
  backgroundElevated: "#221A22",
  surface: "#221A22",
  surfaceMuted: "#2C222C",
  border: "#3A2E3A",
  borderStrong: "#54445A",
  text: "#F6EEE8",
  textMuted: "#B8ABB8",
  textSubtle: "#8B7D8C",
  primary: "#EE6F55",
  primarySoft: "#3B211F",
  primaryForeground: "#1A0E10",
  secondary: "#D9B8D4",
  success: "#5FBF94",
  successSoft: "#172A22",
  warning: "#E0A24A",
  warningSoft: "#33260F",
  danger: "#F2867D",
  dangerSoft: "#3A1B19",
  like: "#EE6F55",
  pass: "#B8ABB8",
  overlay: "rgba(0, 0, 0, 0.6)",
  photoScrim: "rgba(0, 0, 0, 0.6)"
};

export type Theme = {
  colors: Colors;
  spacing: typeof spacing;
  radii: typeof radii;
  fontSizes: typeof fontSizes;
  fontWeights: typeof fontWeights;
  typography: typeof typography;
  shadows: typeof shadows;
  isDark: boolean;
};

function buildTheme(isDark: boolean): Theme {
  return {
    colors: isDark ? darkColors : lightColors,
    spacing,
    radii,
    fontSizes,
    fontWeights,
    typography,
    shadows,
    isDark
  };
}

const ThemeContext = createContext<Theme>(buildTheme(false));

export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme();
  const theme = useMemo(() => buildTheme(scheme === "dark"), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
