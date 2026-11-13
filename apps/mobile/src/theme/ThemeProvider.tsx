import React, { createContext, useContext, useMemo } from "react";
import { Platform, useColorScheme } from "react-native";
import { darkPalette, lightPalette, radii, spacing, type Palette } from "./palette";

export type Theme = {
  dark: boolean;
  colors: Palette;
  spacing: typeof spacing;
  radii: typeof radii;
  displayFont: string | undefined;
};

const ThemeContext = createContext<Theme | null>(null);

const displayFont = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const theme = useMemo<Theme>(() => {
    const dark = scheme === "dark";
    return { dark, colors: dark ? darkPalette : lightPalette, spacing, radii, displayFont };
  }, [scheme]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  const theme = useContext(ThemeContext);
  if (!theme) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return theme;
}

/** Memoized style factory bound to the active theme. */
export function useThemedStyles<T>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => factory(theme), [factory, theme]);
}
