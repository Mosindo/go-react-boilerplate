import React, { createContext, useContext, useMemo } from "react";
import { useColorScheme } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primaryPressed: string;
  onPrimary: string;
  primarySoft: string;
  like: string;
  pass: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  bubbleMine: string;
  onBubbleMine: string;
  bubbleTheirs: string;
};

const light: Palette = {
  background: "#FBF6F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F3EAE3",
  border: "#E7DAD0",
  text: "#2B1B1A",
  textMuted: "#6F5C57",
  textSubtle: "#9A8780",
  primary: "#B8472F",
  primaryPressed: "#9C3A25",
  onPrimary: "#FFFFFF",
  primarySoft: "#F8E4DC",
  like: "#C2412B",
  pass: "#6F5C57",
  success: "#2F7D5B",
  danger: "#B3261E",
  dangerSoft: "#FBE9E7",
  overlay: "rgba(32, 18, 16, 0.55)",
  bubbleMine: "#B8472F",
  onBubbleMine: "#FFFFFF",
  bubbleTheirs: "#F0E5DD"
};

const dark: Palette = {
  background: "#171110",
  surface: "#231A18",
  surfaceMuted: "#2D221F",
  border: "#3A2C28",
  text: "#F7EDE8",
  textMuted: "#C2AFA7",
  textSubtle: "#8F7D76",
  primary: "#E0694D",
  primaryPressed: "#C9573C",
  onPrimary: "#1B0F0C",
  primarySoft: "#3B2019",
  like: "#EE7A5F",
  pass: "#C2AFA7",
  success: "#58B58C",
  danger: "#F2867D",
  dangerSoft: "#3E1E1B",
  overlay: "rgba(0, 0, 0, 0.65)",
  bubbleMine: "#E0694D",
  onBubbleMine: "#1B0F0C",
  bubbleTheirs: "#33261F"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 24, xl: 32, xxl: 48 } as const;
export const radii = { sm: 8, md: 14, lg: 22, xl: 32, pill: 999 } as const;
export const fontSizes = { xs: 12, sm: 14, md: 16, lg: 18, xl: 24, xxl: 32 } as const;

export type Theme = { dark: boolean; colors: Palette };

const ThemeContext = createContext<Theme>({ dark: false, colors: light });

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const scheme = useColorScheme();
  const value = useMemo<Theme>(
    () => (scheme === "dark" ? { dark: true, colors: dark } : { dark: false, colors: light }),
    [scheme]
  );
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  return useContext(ThemeContext);
}
