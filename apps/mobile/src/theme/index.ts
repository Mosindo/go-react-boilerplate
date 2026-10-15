import { useColorScheme, type ViewStyle } from "react-native";
import { darkPalette, lightPalette, type Palette } from "./tokens";

export * from "./tokens";

export function useTheme(): Palette {
  return useColorScheme() === "dark" ? darkPalette : lightPalette;
}

/** Soft elevation that reads well on both palettes. */
export function elevation(palette: Palette, level: 1 | 2 = 1): ViewStyle {
  return {
    shadowColor: palette.isDark ? "#000000" : "#3A2230",
    shadowOpacity: palette.isDark ? 0.4 : level === 1 ? 0.08 : 0.16,
    shadowRadius: level === 1 ? 14 : 26,
    shadowOffset: { width: 0, height: level === 1 ? 6 : 12 },
    elevation: level === 1 ? 3 : 8
  };
}
