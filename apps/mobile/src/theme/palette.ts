export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  textOnPrimary: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  accent: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  bubbleMine: string;
  bubbleTheirs: string;
  isDark: boolean;
};

// "Sillage" identity: warm ivory + terracotta, with a deep plum accent.
export const lightPalette: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  border: "#E8DDD2",
  text: "#2A1F2D",
  textMuted: "#6F6470",
  textOnPrimary: "#FFFFFF",
  primary: "#C8472B",
  primaryPressed: "#A93920",
  primarySoft: "#FBE6DF",
  accent: "#4A2545",
  success: "#1F7A4D",
  successSoft: "#E3F4EA",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  overlay: "rgba(26, 17, 28, 0.55)",
  bubbleMine: "#C8472B",
  bubbleTheirs: "#F0E7DD",
  isDark: false
};

export const darkPalette: Palette = {
  background: "#16121A",
  surface: "#211B27",
  surfaceMuted: "#2B2432",
  border: "#3A3142",
  text: "#F6EFE9",
  textMuted: "#B2A6B3",
  textOnPrimary: "#FFFFFF",
  primary: "#E2603F",
  primaryPressed: "#C8472B",
  primarySoft: "#3D2220",
  accent: "#D9A7D3",
  success: "#5FCB92",
  successSoft: "#17301F",
  danger: "#F28B82",
  dangerSoft: "#3B1E1C",
  overlay: "rgba(0, 0, 0, 0.65)",
  bubbleMine: "#C8472B",
  bubbleTheirs: "#2F2736",
  isDark: true
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 12, lg: 20, xl: 28, pill: 999 } as const;
export const fontSizes = { xs: 12, sm: 14, md: 16, lg: 20, xl: 26, xxl: 34 } as const;
