import React, { createContext, useContext, useMemo, type ReactNode } from "react";
import { useColorScheme } from "react-native";
import { darkColors, lightColors, type Palette } from "./tokens";

type ThemeValue = {
  scheme: "light" | "dark";
  colors: Palette;
};

const ThemeContext = createContext<ThemeValue>({ scheme: "light", colors: lightColors });

/** Follows the system appearance; every themed component reads colors from here. */
export function ThemeProvider({ children, forceScheme }: { children: ReactNode; forceScheme?: "light" | "dark" }) {
  const system = useColorScheme();
  const scheme = forceScheme ?? (system === "dark" ? "dark" : "light");
  const value = useMemo<ThemeValue>(() => ({ scheme, colors: scheme === "dark" ? darkColors : lightColors }), [scheme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  return useContext(ThemeContext);
}
