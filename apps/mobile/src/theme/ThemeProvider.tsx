import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { DarkTheme, DefaultTheme, type Theme as NavigationTheme } from "@react-navigation/native";
import { darkColors, lightColors, type ColorTokens } from "./tokens";

export type AppTheme = {
  colors: ColorTokens;
  isDark: boolean;
  navigationTheme: NavigationTheme;
};

function buildTheme(isDark: boolean): AppTheme {
  const colors = isDark ? darkColors : lightColors;
  const base = isDark ? DarkTheme : DefaultTheme;
  return {
    colors,
    isDark,
    navigationTheme: {
      ...base,
      colors: {
        ...base.colors,
        primary: colors.primary,
        background: colors.background,
        card: colors.background,
        text: colors.text,
        border: colors.border,
        notification: colors.primary
      }
    }
  };
}

const lightTheme = buildTheme(false);
const darkTheme = buildTheme(true);

const ThemeContext = createContext<AppTheme>(lightTheme);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const theme = scheme === "dark" ? darkTheme : lightTheme;
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): AppTheme {
  return useContext(ThemeContext);
}

/** Memoises a style factory (usually returning `StyleSheet.create(...)`) against the current colour tokens. Declare `factory` at module level. */
export function useThemedStyles<T>(factory: (colors: ColorTokens) => T): T {
  const { colors } = useTheme();
  return useMemo(() => factory(colors), [colors, factory]);
}
