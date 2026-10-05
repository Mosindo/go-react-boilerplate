import { createContext, createElement, useContext, useMemo, type ReactNode } from "react";
import { StyleSheet, useColorScheme } from "react-native";
import { darkColors, lightColors, type Colors } from "./tokens";

export * from "./tokens";

type Theme = { colors: Colors; isDark: boolean };

const ThemeContext = createContext<Theme>({ colors: lightColors, isDark: false });

/** Follows the system appearance. The product has no manual theme switch on purpose: one less setting. */
export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => (scheme === "dark" ? { colors: darkColors, isDark: true } : { colors: lightColors, isDark: false }),
    [scheme]
  );
  return createElement(ThemeContext.Provider, { value }, children);
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Builds a themed StyleSheet once per colour scheme: `const styles = useStyles(makeStyles)`. */
export function useStyles<T extends StyleSheet.NamedStyles<T>>(factory: (colors: Colors) => T): T {
  const { colors } = useTheme();
  // The factory is a module-level function, so only the colours can change between renders.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => StyleSheet.create(factory(colors)), [colors]);
}
