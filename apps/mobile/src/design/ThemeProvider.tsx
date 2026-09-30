import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";
import { buildTheme, type Theme } from "./theme";

const ThemeContext = createContext<Theme>(buildTheme(false));

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const theme = useMemo(() => buildTheme(scheme === "dark"), [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

/** Memoised themed styles. Pass a module-level factory so it stays stable. */
export function useStyles<T>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => factory(theme), [factory, theme]);
}
