import * as SecureStore from "expo-secure-store";
import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { StyleSheet, useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primarySoft: string;
  onPrimary: string;
  like: string;
  danger: string;
  dangerSoft: string;
  success: string;
  overlay: string;
  scrim: string;
};

const light: Palette = {
  background: "#FBF7F4",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE6",
  border: "#E9DFD8",
  text: "#2A1F1D",
  textMuted: "#6F5F59",
  primary: "#C93A55",
  primarySoft: "#FBE6EA",
  onPrimary: "#FFFFFF",
  like: "#1F8A63",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  success: "#1F8A63",
  overlay: "rgba(24, 14, 12, 0.45)",
  scrim: "rgba(24, 14, 12, 0.55)"
};

const dark: Palette = {
  background: "#14100F",
  surface: "#1E1816",
  surfaceMuted: "#2A2220",
  border: "#3A2F2C",
  text: "#F6EEEA",
  textMuted: "#B3A39D",
  primary: "#F0617A",
  primarySoft: "#3A1D24",
  onPrimary: "#1A0A0E",
  like: "#4CC79B",
  danger: "#FF8A80",
  dangerSoft: "#3B1E1B",
  success: "#4CC79B",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.6)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;

export type ThemeMode = "system" | "light" | "dark";

export type Theme = {
  colors: Palette;
  isDark: boolean;
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
};

const MODE_KEY = "app.theme.mode";
const ThemeContext = createContext<Theme | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const [mode, setModeState] = useState<ThemeMode>("system");

  useEffect(() => {
    SecureStore.getItemAsync(MODE_KEY)
      .then((stored) => {
        if (stored === "light" || stored === "dark" || stored === "system") {
          setModeState(stored);
        }
      })
      .catch(() => undefined);
  }, []);

  const setMode = useCallback((next: ThemeMode) => {
    setModeState(next);
    SecureStore.setItemAsync(MODE_KEY, next).catch(() => undefined);
  }, []);

  const isDark = mode === "dark" || (mode === "system" && system === "dark");
  const value = useMemo<Theme>(() => ({ colors: isDark ? dark : light, isDark, mode, setMode }), [isDark, mode, setMode]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  const theme = useContext(ThemeContext);
  if (!theme) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return theme;
}

/** Builds a StyleSheet once per palette change. */
export function useStyles<T extends StyleSheet.NamedStyles<T>>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- factories are module-level pure functions
  return useMemo(() => StyleSheet.create(factory(colors)), [colors]);
}

export const APP_NAME = process.env.EXPO_PUBLIC_APP_NAME?.trim() || "Mellow";
