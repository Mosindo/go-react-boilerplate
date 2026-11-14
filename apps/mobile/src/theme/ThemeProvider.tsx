import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { dark, light, type Palette } from "./palette";

type Theme = { colors: Palette; scheme: "light" | "dark" };

const ThemeContext = createContext<Theme>({ colors: light, scheme: "light" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const value = useMemo<Theme>(() => ({ colors: scheme === "dark" ? dark : light, scheme }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised themed StyleSheet factory: `const styles = useStyles(makeStyles)`. */
export function useStyles<T>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  return useMemo(() => factory(colors), [colors, factory]);
}
