import { useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  onPrimary: string;
  accent: string;
  danger: string;
  success: string;
  overlay: string;
};

export const lightPalette: Palette = {
  background: "#FBF5EE",
  surface: "#FFFFFF",
  surfaceAlt: "#F3E9DE",
  text: "#2A1722",
  textMuted: "#6F5C66",
  border: "#E6D8C9",
  primary: "#7A2E5B",
  onPrimary: "#FFFFFF",
  accent: "#C75B2F",
  danger: "#B3261E",
  success: "#2E7D5B",
  overlay: "rgba(42, 23, 34, 0.55)"
};

export const darkPalette: Palette = {
  background: "#170F14",
  surface: "#231820",
  surfaceAlt: "#2E2029",
  text: "#F6ECE4",
  textMuted: "#B9A5AE",
  border: "#3A2A33",
  primary: "#D58AB8",
  onPrimary: "#2A1722",
  accent: "#E58A62",
  danger: "#F2827A",
  success: "#6CCBA0",
  overlay: "rgba(0, 0, 0, 0.65)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 14, lg: 22, pill: 999 } as const;

export function useTheme(): { colors: Palette; isDark: boolean } {
  const scheme = useColorScheme();
  const isDark = scheme === "dark";
  return { colors: isDark ? darkPalette : lightPalette, isDark };
}
