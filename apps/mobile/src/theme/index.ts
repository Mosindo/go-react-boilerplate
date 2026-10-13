import type React from "react";
import { createContext, createElement, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkColors, lightColors, type Colors } from "../shared/ui/tokens";

export type ThemeValue = {
  scheme: "light" | "dark";
  colors: Colors;
};

const ThemeContext = createContext<ThemeValue>({ scheme: "light", colors: lightColors });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const system = useColorScheme();
  const value = useMemo<ThemeValue>(
    () => (system === "dark" ? { scheme: "dark", colors: darkColors } : { scheme: "light", colors: lightColors }),
    [system]
  );
  return createElement(ThemeContext.Provider, { value }, children);
}

export function useTheme(): ThemeValue {
  return useContext(ThemeContext);
}
