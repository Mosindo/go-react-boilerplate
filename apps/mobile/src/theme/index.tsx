import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkColors, lightColors, type ThemeColors } from "./tokens";

export * from "./tokens";

type Theme = { colors: ThemeColors; isDark: boolean };

const ThemeContext = createContext<Theme>({ colors: lightColors, isDark: false });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => ({ colors: scheme === "dark" ? darkColors : lightColors, isDark: scheme === "dark" }),
    [scheme]
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
