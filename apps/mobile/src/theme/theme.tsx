import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";

export type Palette = {
  bg: string;
  surface: string;
  surfaceAlt: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  accent: string;
  border: string;
  danger: string;
  success: string;
  overlay: string;
};

// Alba palette: deep plum + apricot on warm paper tones. Text/background pairs meet WCAG AA.
const light: Palette = {
  bg: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceAlt: "#F2EBE6",
  text: "#241B2B",
  textMuted: "#6B5F73",
  primary: "#5B2A63",
  onPrimary: "#FFFFFF",
  accent: "#F4A261",
  border: "#E3D9D2",
  danger: "#B3261E",
  success: "#2E7D4F",
  overlay: "rgba(36, 27, 43, 0.55)",
};

const dark: Palette = {
  bg: "#16121A",
  surface: "#211B27",
  surfaceAlt: "#2C2434",
  text: "#F4EEF6",
  textMuted: "#B6A9BE",
  primary: "#C79BD1",
  onPrimary: "#1A1020",
  accent: "#F4A261",
  border: "#3A3043",
  danger: "#FF8A80",
  success: "#6FCF97",
  overlay: "rgba(0, 0, 0, 0.6)",
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 12, lg: 20, pill: 999 } as const;

type ThemeValue = { colors: Palette; scheme: "light" | "dark" };

const ThemeContext = createContext<ThemeValue>({ colors: light, scheme: "light" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme: ThemeValue["scheme"] = useColorScheme() === "dark" ? "dark" : "light";
  const value = useMemo(() => ({ colors: scheme === "dark" ? dark : light, scheme }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  return useContext(ThemeContext);
}
