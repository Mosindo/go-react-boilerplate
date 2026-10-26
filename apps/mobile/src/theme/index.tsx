import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";

const light = {
  bg: "#FBF6F1",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EBE3",
  text: "#2A1F2D",
  textMuted: "#6B5F6E",
  border: "#E6DACF",
  primary: "#B84A33",
  primaryPressed: "#9E3D29",
  onPrimary: "#FFFFFF",
  primarySoft: "#F8E6E0",
  success: "#2B7455",
  successSoft: "#E1F1E9",
  danger: "#B3261E",
  dangerSoft: "#FBE7E5",
  overlay: "rgba(32, 22, 35, 0.55)",
  like: "#2B7455",
  pass: "#6B5F6E"
};

const dark: typeof light = {
  bg: "#161118",
  surface: "#211A24",
  surfaceAlt: "#2D2431",
  text: "#F6EEE8",
  textMuted: "#B2A6B6",
  border: "#3C3142",
  primary: "#E2735B",
  primaryPressed: "#EE8A74",
  onPrimary: "#1E0F12",
  primarySoft: "#3A2326",
  success: "#58C79A",
  successSoft: "#1E3A2F",
  danger: "#F2918A",
  dangerSoft: "#3F2220",
  overlay: "rgba(0, 0, 0, 0.65)",
  like: "#58C79A",
  pass: "#B2A6B6"
};

export type Theme = typeof light & { scheme: "light" | "dark" };

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;

const ThemeContext = createContext<Theme>({ ...light, scheme: "light" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const value = useMemo<Theme>(() => ({ ...(scheme === "dark" ? dark : light), scheme }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
