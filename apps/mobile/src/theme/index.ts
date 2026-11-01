import { useColorScheme } from "react-native";

/** Identite visuelle "Alma": creme chaleureux, aubergine profonde, rose terracotta. */
const light = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  text: "#2A1F2D",
  textMuted: "#7A6C7D",
  primary: "#C2475B",
  primaryPressed: "#9E3347",
  onPrimary: "#FFFFFF",
  accent: "#E3A857",
  border: "#EADFD6",
  danger: "#C23B3B",
  success: "#3E8E6B",
  overlay: "rgba(42,31,45,0.55)",
  like: "#3E8E6B",
  pass: "#7A6C7D",
};

const dark: typeof light = {
  background: "#17121A",
  surface: "#231C27",
  surfaceAlt: "#2D2432",
  text: "#F5EDF2",
  textMuted: "#A99BAE",
  primary: "#E0627A",
  primaryPressed: "#C94C65",
  onPrimary: "#1B1119",
  accent: "#E3A857",
  border: "#362C3B",
  danger: "#EF6B6B",
  success: "#5CB88F",
  overlay: "rgba(0,0,0,0.65)",
  like: "#5CB88F",
  pass: "#A99BAE",
};

export type Colors = typeof light;

export function useColors(): Colors {
  return useColorScheme() === "dark" ? dark : light;
}

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
