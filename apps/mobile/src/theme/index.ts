import { useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primaryText: string;
  primarySoft: string;
  accent: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
};

export const light: Palette = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE5",
  border: "#E8DDD3",
  text: "#2A2024",
  textMuted: "#6F6168",
  primary: "#C8431F",
  primaryText: "#FFFFFF",
  primarySoft: "#FBE7DF",
  accent: "#7A3E65",
  success: "#2F7D5B",
  danger: "#B3261E",
  dangerSoft: "#FBE9E7",
  overlay: "rgba(26, 17, 21, 0.55)",
  scrim: "rgba(20, 12, 16, 0.62)"
};

export const dark: Palette = {
  background: "#171117",
  surface: "#221A22",
  surfaceAlt: "#2D232D",
  border: "#3B2F3B",
  text: "#F6EEF0",
  textMuted: "#B9A9B2",
  primary: "#F26B45",
  primaryText: "#1B0F0C",
  primarySoft: "#3A2019",
  accent: "#D49BC1",
  success: "#5FC49B",
  danger: "#FF8A80",
  dangerSoft: "#3D1F1D",
  overlay: "rgba(0, 0, 0, 0.65)",
  scrim: "rgba(0, 0, 0, 0.6)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 12, lg: 20, xl: 28, pill: 999 } as const;

export function usePalette(): Palette {
  return useColorScheme() === "dark" ? dark : light;
}
