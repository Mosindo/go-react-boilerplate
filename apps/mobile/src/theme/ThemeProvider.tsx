import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useColorScheme } from "react-native";
import { readItem, writeItem } from "../utils/storage";
import { palettes, type ColorScheme, type Palette } from "./palette";

export type ThemePreference = "system" | "light" | "dark";

type ThemeValue = {
  scheme: ColorScheme;
  colors: Palette;
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
};

const PREFERENCE_KEY = "lueur.theme";
const ThemeContext = createContext<ThemeValue | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const [preference, setPreferenceState] = useState<ThemePreference>("system");

  useEffect(() => {
    void readItem(PREFERENCE_KEY).then((value) => {
      if (value === "light" || value === "dark" || value === "system") {
        setPreferenceState(value);
      }
    });
  }, []);

  const setPreference = useCallback((next: ThemePreference) => {
    setPreferenceState(next);
    void writeItem(PREFERENCE_KEY, next);
  }, []);

  const scheme: ColorScheme = preference === "system" ? (system === "dark" ? "dark" : "light") : preference;
  const value = useMemo<ThemeValue>(
    () => ({ scheme, colors: palettes[scheme], preference, setPreference }),
    [scheme, preference, setPreference]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  const value = useContext(ThemeContext);
  if (!value) {
    throw new Error("useTheme must be used inside ThemeProvider");
  }
  return value;
}

/** Memoised themed StyleSheet factory. */
export function useStyles<T>(factory: (colors: Palette, scheme: ColorScheme) => T): T {
  const { colors, scheme } = useTheme();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => factory(colors, scheme), [colors, scheme]);
}
