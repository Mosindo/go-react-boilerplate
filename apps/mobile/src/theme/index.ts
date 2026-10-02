import { useColorScheme } from "react-native";

export interface Colors {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primaryPressed: string;
  onPrimary: string;
  primarySoft: string;
  accent: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  like: string;
  pass: string;
}

// Warm paper + terracotta, with a plum accent. Dark mode keeps the same hues, lower luminance.
const light: Colors = {
  background: "#FBF7F4",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE7",
  border: "#E7DDD6",
  text: "#2A1F2D",
  textMuted: "#75666E",
  primary: "#C2503A",
  primaryPressed: "#A33F2B",
  onPrimary: "#FFFFFF",
  primarySoft: "#F8E3DC",
  accent: "#5B3A6B",
  success: "#2F8F63",
  danger: "#B3261E",
  dangerSoft: "#F9DEDC",
  overlay: "rgba(26, 17, 28, 0.55)",
  like: "#2F8F63",
  pass: "#75666E",
};

const dark: Colors = {
  background: "#171216",
  surface: "#241C23",
  surfaceAlt: "#2E242D",
  border: "#3B2F3A",
  text: "#F6EFEA",
  textMuted: "#B2A3AB",
  primary: "#E2705A",
  primaryPressed: "#EE8A76",
  onPrimary: "#1C1217",
  primarySoft: "#4A2A26",
  accent: "#B891CC",
  success: "#5CC795",
  danger: "#F2B8B5",
  dangerSoft: "#4F2623",
  overlay: "rgba(0, 0, 0, 0.65)",
  like: "#5CC795",
  pass: "#B2A3AB",
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 14, lg: 22, pill: 999 } as const;

export interface Theme {
  colors: Colors;
  isDark: boolean;
}

export function useTheme(): Theme {
  const scheme = useColorScheme();
  const isDark = scheme === "dark";
  return { colors: isDark ? dark : light, isDark };
}
