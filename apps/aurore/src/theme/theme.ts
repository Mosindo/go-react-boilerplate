import { useColorScheme } from "react-native";

const light = {
  background: "#FBF7F4",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE7",
  text: "#2A1F2D",
  textMuted: "#6B5E6E",
  border: "#E6DBD5",
  primary: "#C2412B",
  onPrimary: "#FFFFFF",
  primarySoft: "#FBE6E0",
  accent: "#7A3E65",
  accentSoft: "#F1E4EC",
  success: "#2B7A57",
  successSoft: "#E1F2EA",
  danger: "#B3261E",
  dangerSoft: "#FBE4E2",
  overlay: "rgba(26, 17, 28, 0.55)",
  like: "#2B7A57",
  pass: "#6B5E6E"
};

const dark: typeof light = {
  background: "#16111A",
  surface: "#211A26",
  surfaceAlt: "#2B2231",
  text: "#F6EEE9",
  textMuted: "#B9ABBC",
  border: "#3A2F41",
  primary: "#F2785C",
  onPrimary: "#1B0D09",
  primarySoft: "#3C211C",
  accent: "#D7A3C4",
  accentSoft: "#34243A",
  success: "#6CCB9C",
  successSoft: "#1D3328",
  danger: "#FF8A80",
  dangerSoft: "#3D1F1D",
  overlay: "rgba(0, 0, 0, 0.65)",
  like: "#6CCB9C",
  pass: "#B9ABBC"
};

export type Theme = typeof light & { isDark: boolean };

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 12, lg: 20, xl: 28, pill: 999 } as const;
export const MIN_TOUCH = 44;

export function useTheme(): Theme {
  const scheme = useColorScheme();
  return scheme === "dark" ? { ...dark, isDark: true } : { ...light, isDark: false };
}
