import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useColorScheme } from "react-native";
import { palettes, type ColorScheme, type Palette } from "./tokens";
import { loadPreference, savePreference } from "../lib/preferencesStore";

export type ThemePreference = "system" | ColorScheme;

type ThemeValue = {
  colors: Palette;
  scheme: ColorScheme;
  preference: ThemePreference;
  setPreference: (value: ThemePreference) => void;
};

const ThemeContext = createContext<ThemeValue | null>(null);
const STORAGE_KEY = "theme";

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const [preference, setPreferenceState] = useState<ThemePreference>("system");

  useEffect(() => {
    void loadPreference(STORAGE_KEY).then((stored) => {
      if (stored === "light" || stored === "dark" || stored === "system") {
        setPreferenceState(stored);
      }
    });
  }, []);

  const setPreference = useCallback((value: ThemePreference) => {
    setPreferenceState(value);
    void savePreference(STORAGE_KEY, value);
  }, []);

  const scheme: ColorScheme = preference === "system" ? (system === "dark" ? "dark" : "light") : preference;
  const value = useMemo(() => ({ colors: palettes[scheme], scheme, preference, setPreference }), [scheme, preference, setPreference]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return ctx;
}

/** Memoised StyleSheet factory bound to the active palette. */
export function useStyles<T>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  return useMemo(() => factory(colors), [colors, factory]);
}
