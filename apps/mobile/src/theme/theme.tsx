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
  onPrimary: string;
  accent: string;
  success: string;
  danger: string;
  overlay: string;
  like: string;
  pass: string;
};

// "Ember": warm paper tones with a terracotta accent. Original palette, not derived from any dating brand.
const light: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  text: "#2A211E",
  textMuted: "#7A6C65",
  border: "#E7DCD1",
  primary: "#D9532F",
  onPrimary: "#FFFFFF",
  accent: "#5B2A4E",
  success: "#2E8B57",
  danger: "#B3261E",
  overlay: "rgba(30, 20, 16, 0.55)",
  like: "#2E8B57",
  pass: "#8A7B73"
};

const dark: Palette = {
  background: "#171213",
  surface: "#241C1E",
  surfaceAlt: "#2F2527",
  text: "#F6EEE8",
  textMuted: "#B2A29A",
  border: "#3A2E30",
  primary: "#F0684A",
  onPrimary: "#1A0F0C",
  accent: "#D9A4CB",
  success: "#5CC48B",
  danger: "#FF8A80",
  overlay: "rgba(0, 0, 0, 0.65)",
  like: "#5CC48B",
  pass: "#A8998F"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 14, lg: 24, pill: 999 } as const;

type ThemeValue = { colors: Palette; isDark: boolean };
const ThemeContext = createContext<ThemeValue>({ colors: light, isDark: false });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<ThemeValue>(() => ({ colors: scheme === "dark" ? dark : light, isDark: scheme === "dark" }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  return useContext(ThemeContext);
}
