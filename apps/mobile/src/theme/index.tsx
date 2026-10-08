import React, { createContext, useContext, useMemo } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./tokens";

export * from "./tokens";

export type Theme = { colors: Palette; scheme: "light" | "dark" };

const ThemeContext = createContext<Theme>({ colors: lightPalette, scheme: "light" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const value = useMemo<Theme>(() => ({ scheme, colors: scheme === "dark" ? darkPalette : lightPalette }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

type Styles = StyleSheet.NamedStyles<Record<string, unknown>>;

/** Builds a StyleSheet once per palette: `const styles = useStyles(createStyles)`. */
export function useStyles<T extends Styles | { [key: string]: object }>(factory: (colors: Palette) => T): T {
  const { colors } = useTheme();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => StyleSheet.create(factory(colors) as never) as unknown as T, [colors]);
}
