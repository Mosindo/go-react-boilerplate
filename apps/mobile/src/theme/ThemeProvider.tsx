import React, { createContext, useContext, useMemo, type ReactNode } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { dark, light, radii, spacing, typography, type Palette } from "./tokens";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
  spacing: typeof spacing;
  radii: typeof radii;
  typography: typeof typography;
  shadow: (elevation: 1 | 2 | 3) => object;
};

function buildTheme(scheme: "light" | "dark"): Theme {
  const colors = scheme === "dark" ? dark : light;
  return {
    scheme,
    colors,
    spacing,
    radii,
    typography,
    shadow: (elevation) => ({
      shadowColor: colors.shadow,
      shadowOpacity: scheme === "dark" ? 0.5 : 0.08 + elevation * 0.03,
      shadowRadius: 6 + elevation * 6,
      shadowOffset: { width: 0, height: 2 + elevation * 3 },
      elevation: elevation * 3
    })
  };
}

const ThemeContext = createContext<Theme>(buildTheme("light"));

export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const theme = useMemo(() => buildTheme(scheme), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoized, theme-aware StyleSheet factory. */
export function makeStyles<T extends StyleSheet.NamedStyles<T>>(factory: (theme: Theme) => T) {
  return function useStyles(): T {
    const theme = useTheme();
    return useMemo(() => StyleSheet.create(factory(theme)), [theme]);
  };
}
