import { useMemo } from "react";
import { useColorScheme } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./tokens";

export * from "./tokens";

export type ThemeMode = "light" | "dark";

export function useThemeMode(): ThemeMode {
  return useColorScheme() === "dark" ? "dark" : "light";
}

export function useColors(): Palette {
  const mode = useThemeMode();
  return mode === "dark" ? darkPalette : lightPalette;
}

/** Builds a StyleSheet-like object from the current palette, memoised per scheme. */
export function useThemedStyles<T>(factory: (colors: Palette) => T): T {
  const colors = useColors();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- factory is intentionally a module-level function
  return useMemo(() => factory(colors), [colors]);
}
