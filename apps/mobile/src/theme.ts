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
  danger: string;
  success: string;
  overlay: string;
};

const light: Palette = {
  background: "#FBF6F1",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EAE1",
  text: "#2A1F22",
  textMuted: "#6F6166",
  border: "#E6D9CD",
  primary: "#D9482B",
  primaryText: "#FFFFFF",
  danger: "#B3261E",
  success: "#2E7D4F",
  overlay: "rgba(30, 18, 22, 0.55)"
};

const dark: Palette = {
  background: "#171113",
  surface: "#241B1E",
  surfaceAlt: "#312529",
  text: "#F7EEE9",
  textMuted: "#B8A8AD",
  border: "#3D2F34",
  primary: "#F0613F",
  primaryText: "#1A0F0B",
  danger: "#FF8A80",
  success: "#6FCF97",
  overlay: "rgba(0, 0, 0, 0.65)"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 8, md: 14, lg: 24, pill: 999 } as const;

export function useTheme(): Palette {
  return useColorScheme() === "dark" ? dark : light;
}
