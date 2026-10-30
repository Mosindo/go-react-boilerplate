import { useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  primaryText: string;
  primarySoft: string;
  accent: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  photoScrim: string;
};

const light: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  text: "#2A211C",
  textMuted: "#6E625A",
  border: "#E8DED3",
  primary: "#C2411F",
  primaryText: "#FFFFFF",
  primarySoft: "#FBE6DE",
  accent: "#5B2A4F",
  success: "#2E7D4F",
  danger: "#B3261E",
  dangerSoft: "#FBE4E2",
  overlay: "rgba(26, 18, 14, 0.55)",
  photoScrim: "rgba(20, 12, 8, 0.62)"
};

const dark: Palette = {
  background: "#17120F",
  surface: "#221B17",
  surfaceAlt: "#2C241F",
  text: "#F5EEE8",
  textMuted: "#B9ABA0",
  border: "#3A302A",
  primary: "#F2845C",
  primaryText: "#1A0F0A",
  primarySoft: "#3A241B",
  accent: "#D9A6C9",
  success: "#7FD1A0",
  danger: "#FF8A80",
  dangerSoft: "#3D1F1D",
  overlay: "rgba(0, 0, 0, 0.65)",
  photoScrim: "rgba(8, 5, 3, 0.7)"
};

export function useTheme(): { colors: Palette; isDark: boolean } {
  const scheme = useColorScheme();
  const isDark = scheme === "dark";
  return { colors: isDark ? dark : light, isDark };
}

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;
export const MIN_TOUCH = 44;
