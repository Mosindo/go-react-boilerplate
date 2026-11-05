import React, { createContext, useContext, useMemo, type ReactNode } from "react";
import { useColorScheme } from "react-native";

// "Ember": warm, adult, minimal. One accent (terracotta), generous contrast.
const light = {
  mode: "light" as const,
  bg: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE6",
  border: "#E7DDD5",
  text: "#2A1F1D",
  textMuted: "#7A6A66",
  primary: "#C9503C",
  primaryPressed: "#B04230",
  onPrimary: "#FFFFFF",
  primarySoft: "#F8E3DD",
  success: "#2F7D5B",
  danger: "#B42318",
  dangerSoft: "#FBE4E1",
  overlay: "rgba(26, 17, 15, 0.55)",
  bubbleMine: "#C9503C",
  bubbleTheirs: "#EFE6DF"
};

export type Palette = Omit<typeof light, "mode"> & { mode: "light" | "dark" };

const dark: Palette = {
  mode: "dark",
  bg: "#15100F",
  surface: "#211918",
  surfaceAlt: "#2C2220",
  border: "#3A2E2B",
  text: "#F6EDE9",
  textMuted: "#B1A09A",
  primary: "#E8765F",
  primaryPressed: "#D9644D",
  onPrimary: "#1B0F0C",
  primarySoft: "#3B2420",
  success: "#5FBF95",
  danger: "#F27C70",
  dangerSoft: "#3D2220",
  overlay: "rgba(0, 0, 0, 0.65)",
  bubbleMine: "#C9503C",
  bubbleTheirs: "#2E2422"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 12, lg: 20, xl: 28, pill: 999 } as const;

const ThemeContext = createContext<Palette>(light);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme();
  const palette = useMemo<Palette>(() => (scheme === "dark" ? dark : light), [scheme]);
  return <ThemeContext.Provider value={palette}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Palette {
  return useContext(ThemeContext);
}
