import React, { createContext, useContext, useMemo } from "react";
import {
  Platform,
  StyleSheet,
  useColorScheme,
  type ImageStyle,
  type TextStyle,
  type ViewStyle
} from "react-native";
import {
  CONTENT_MAX_WIDTH,
  MIN_TOUCH,
  darkPalette,
  lightPalette,
  radius,
  spacing,
  type Palette
} from "./palette";

export type Theme = {
  scheme: "light" | "dark";
  colors: Palette;
  spacing: typeof spacing;
  radius: typeof radius;
  minTouch: number;
  contentMaxWidth: number;
  fontFamily: string | undefined;
};

const fontFamily = Platform.select({
  ios: "Avenir Next",
  android: "sans-serif-medium",
  default: "system-ui"
});

function buildTheme(scheme: "light" | "dark"): Theme {
  return {
    scheme,
    colors: scheme === "dark" ? darkPalette : lightPalette,
    spacing,
    radius,
    minTouch: MIN_TOUCH,
    contentMaxWidth: CONTENT_MAX_WIDTH,
    fontFamily
  };
}

const themes = { light: buildTheme("light"), dark: buildTheme("dark") };
const ThemeContext = createContext<Theme>(themes.light);

/** Follows the system appearance. */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  return <ThemeContext.Provider value={themes[scheme]}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}

type Styles = Record<string, ViewStyle | TextStyle | ImageStyle>;

/** Builds a hook that returns styles derived from the current theme (memoized per scheme). */
export function createStyles<T extends Styles>(factory: (theme: Theme) => T) {
  return function useStyles(): T {
    const theme = useTheme();
    return useMemo(() => StyleSheet.create(factory(theme)) as unknown as T, [theme]);
  };
}
