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
  overlay: string;
  bubbleMine: string;
  bubbleMineText: string;
  bubbleTheirs: string;
};

const light: Palette = {
  background: "#FBF6F0",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EAE0",
  border: "#E6D9CB",
  text: "#2A1F2D",
  textMuted: "#6F6272",
  primary: "#D9482B",
  primaryText: "#FFFFFF",
  primarySoft: "#FBE3DC",
  accent: "#E9A23B",
  success: "#2F8F5B",
  danger: "#B3261E",
  overlay: "rgba(24, 14, 28, 0.55)",
  bubbleMine: "#D9482B",
  bubbleMineText: "#FFFFFF",
  bubbleTheirs: "#F0E6DB"
};

const dark: Palette = {
  background: "#171219",
  surface: "#231C27",
  surfaceAlt: "#2E2533",
  border: "#3A3040",
  text: "#F6EFE9",
  textMuted: "#B2A5B6",
  primary: "#F0664A",
  primaryText: "#1B0F0C",
  primarySoft: "#3A2320",
  accent: "#F0B35A",
  success: "#5CC48B",
  danger: "#FF8A80",
  overlay: "rgba(0, 0, 0, 0.65)",
  bubbleMine: "#F0664A",
  bubbleMineText: "#1B0F0C",
  bubbleTheirs: "#2E2533"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 12, lg: 20, pill: 999 } as const;
export const fontSizes = { xs: 12, sm: 14, md: 16, lg: 20, xl: 26, xxl: 34 } as const;

export type Theme = { dark: boolean; colors: Palette };

export function useTheme(): Theme {
  const scheme = useColorScheme();
  const isDark = scheme === "dark";
  return { dark: isDark, colors: isDark ? dark : light };
}
