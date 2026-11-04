import { useColorScheme } from "react-native";

export type Colors = {
  background: string;
  surface: string;
  surfaceMuted: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  accent: string;
  onAccent: string;
  border: string;
  danger: string;
  success: string;
  overlay: string;
  like: string;
  pass: string;
};

const light: Colors = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  text: "#2A1830",
  textMuted: "#6F6074",
  primary: "#4A1D4F",
  onPrimary: "#FFFFFF",
  accent: "#B84A32",
  onAccent: "#FFFFFF",
  border: "#E6DCD2",
  danger: "#B3261E",
  success: "#2E7D5B",
  overlay: "rgba(26,12,30,0.55)",
  like: "#2E7D5B",
  pass: "#8A7A8F",
};

const dark: Colors = {
  background: "#160E19",
  surface: "#231A27",
  surfaceMuted: "#2E2333",
  text: "#F4EBE3",
  textMuted: "#B2A4B7",
  primary: "#E3B7EA",
  onPrimary: "#2A1230",
  accent: "#E5806A",
  onAccent: "#2A1230",
  border: "#3A2D40",
  danger: "#F2827A",
  success: "#6BCB9D",
  overlay: "rgba(0,0,0,0.65)",
  like: "#6BCB9D",
  pass: "#B2A4B7",
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const MIN_TOUCH = 44;

export function useColors(): Colors {
  return useColorScheme() === "dark" ? dark : light;
}

export function useIsDark(): boolean {
  return useColorScheme() === "dark";
}
