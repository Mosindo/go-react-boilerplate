import { useColorScheme } from "react-native";

/** Single source of truth for product identity and design tokens. */
export const BRAND = {
  name: "Lumen",
  tagline: "Meet people worth meeting.",
  promise: "Free, always. No ads, no paywall."
} as const;

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  primarySoft: string;
  accent: string;
  onAccent: string;
  danger: string;
  onDanger: string;
  success: string;
  overlay: string;
  skeleton: string;
  isDark: boolean;
};

export const lightPalette: Palette = {
  background: "#F7F1EA",
  surface: "#FFFFFF",
  surfaceAlt: "#EFE6DB",
  border: "#DCCFC1",
  text: "#2A1B2D",
  textMuted: "#6B5A6E",
  primary: "#B8472F",
  onPrimary: "#FFFFFF",
  primarySoft: "#F6DDD4",
  accent: "#3B1F4A",
  onAccent: "#FFFFFF",
  danger: "#B3261E",
  onDanger: "#FFFFFF",
  success: "#2E7D5B",
  overlay: "rgba(42, 27, 45, 0.55)",
  skeleton: "#E6DACD",
  isDark: false
};

export const darkPalette: Palette = {
  background: "#1A121D",
  surface: "#261A2A",
  surfaceAlt: "#33243A",
  border: "#4A3852",
  text: "#F5ECE4",
  textMuted: "#BBA9BD",
  primary: "#E27D60",
  onPrimary: "#1A121D",
  primarySoft: "#4A2A2A",
  accent: "#C9A7DA",
  onAccent: "#1A121D",
  danger: "#F2867F",
  onDanger: "#1A121D",
  success: "#6CCB9F",
  overlay: "rgba(0, 0, 0, 0.65)",
  skeleton: "#33243A",
  isDark: true
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const hitSlop = 44;

export const typography = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const },
  title: { fontSize: 24, lineHeight: 30, fontWeight: "700" as const },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "600" as const },
  body: { fontSize: 16, lineHeight: 22, fontWeight: "400" as const },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" as const },
  caption: { fontSize: 12, lineHeight: 16, fontWeight: "400" as const }
};

export function useTheme(): Palette {
  return useColorScheme() === "dark" ? darkPalette : lightPalette;
}
