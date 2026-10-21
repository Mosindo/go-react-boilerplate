import { useColorScheme } from "react-native";

export type Palette = {
  scheme: "light" | "dark";
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
  overlay: string;
};

// Warm, adult palette: terracotta primary on a cream / plum-black base.
export const palettes: Record<"light" | "dark", Palette> = {
  light: {
    scheme: "light",
    background: "#FBF7F2",
    surface: "#FFFFFF",
    surfaceAlt: "#F3EBE4",
    border: "#E6DAD0",
    text: "#2B2025",
    textMuted: "#6F6169",
    primary: "#C8452F",
    primaryText: "#FFFFFF",
    primarySoft: "#F9E3DD",
    accent: "#5B2A4A",
    success: "#2F7D5B",
    danger: "#B3261E",
    overlay: "rgba(30, 18, 24, 0.55)"
  },
  dark: {
    scheme: "dark",
    background: "#161115",
    surface: "#221A20",
    surfaceAlt: "#2D232A",
    border: "#3A2E36",
    text: "#F6EEF0",
    textMuted: "#B4A5AD",
    primary: "#EE7560",
    primaryText: "#1B100E",
    primarySoft: "#3F2621",
    accent: "#D9A5C8",
    success: "#6BC49C",
    danger: "#FF8A80",
    overlay: "rgba(0, 0, 0, 0.65)"
  }
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const fontSize = { xs: 12, sm: 14, md: 16, lg: 18, xl: 22, xxl: 30 } as const;
export const MIN_TOUCH = 44;

export function useTheme(): Palette {
  return useColorScheme() === "dark" ? palettes.dark : palettes.light;
}
