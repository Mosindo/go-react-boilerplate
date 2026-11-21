import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { readPreference, writePreference } from "../store/preferencesStore";
import { buildTheme, type ColorScheme, type Theme } from "./tokens";

export type ThemePreference = "system" | ColorScheme;

type ThemeContextValue = {
  theme: Theme;
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);
const PREFERENCE_KEY = "theme";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const [preference, setPreferenceState] = useState<ThemePreference>("system");

  useEffect(() => {
    void readPreference(PREFERENCE_KEY).then((stored) => {
      if (stored === "light" || stored === "dark" || stored === "system") {
        setPreferenceState(stored);
      }
    });
  }, []);

  const setPreference = useCallback((next: ThemePreference) => {
    setPreferenceState(next);
    void writePreference(PREFERENCE_KEY, next);
  }, []);

  const scheme: ColorScheme = preference === "system" ? (system === "dark" ? "dark" : "light") : preference;
  const value = useMemo(() => ({ theme: buildTheme(scheme), preference, setPreference }), [scheme, preference, setPreference]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useThemeContext(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used inside ThemeProvider");
  }
  return ctx;
}

export function useTheme(): Theme {
  return useThemeContext().theme;
}

/** Memoized theme-aware StyleSheet factory. */
export function makeStyles<T extends StyleSheet.NamedStyles<T>>(factory: (theme: Theme) => T): () => T {
  return function useStyles() {
    const theme = useTheme();
    return useMemo(() => StyleSheet.create(factory(theme)), [theme]);
  };
}
