import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";

export type Palette = {
  bg: string;
  surface: string;
  surfaceAlt: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  primaryText: string;
  accent: string;
  danger: string;
  success: string;
  overlay: string;
};

// "Aube": warm paper tones with a terracotta accent. Adult, calm, high contrast.
const light: Palette = {
  bg: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  text: "#2A2320",
  textMuted: "#6B5F58",
  border: "#E4D9CE",
  primary: "#B9442B",
  primaryText: "#FFFFFF",
  accent: "#4F3A57",
  danger: "#B3261E",
  success: "#2E7D4F",
  overlay: "rgba(20,12,8,0.55)",
};

const dark: Palette = {
  bg: "#16110F",
  surface: "#211A17",
  surfaceAlt: "#2B221E",
  text: "#F6EEE7",
  textMuted: "#B8A99D",
  border: "#3A2F29",
  primary: "#E2735A",
  primaryText: "#1A0F0B",
  accent: "#C9A7D4",
  danger: "#F2837B",
  success: "#6FCF97",
  overlay: "rgba(0,0,0,0.65)",
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const font = { xs: 12, sm: 14, md: 16, lg: 20, xl: 28, xxl: 36 } as const;

type Theme = { colors: Palette; isDark: boolean };
const ThemeContext = createContext<Theme>({ colors: light, isDark: false });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => ({ colors: scheme === "dark" ? dark : light, isDark: scheme === "dark" }),
    [scheme],
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
