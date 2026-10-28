export type ThemeColors = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primaryText: string;
  primarySoft: string;
  accent: string;
  like: string;
  pass: string;
  danger: string;
  dangerSoft: string;
  success: string;
  overlay: string;
  bubbleMine: string;
  bubbleMineText: string;
  bubbleTheirs: string;
  skeleton: string;
};

export const lightColors: ThemeColors = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE6",
  border: "#E6DCD3",
  text: "#2A1B2E",
  textMuted: "#6F6272",
  primary: "#6B2D5C",
  primaryText: "#FFFFFF",
  primarySoft: "#F1E1EC",
  accent: "#F26B4F",
  like: "#E8553D",
  pass: "#6F6272",
  danger: "#B42318",
  dangerSoft: "#FDECEA",
  success: "#1B7A4B",
  overlay: "rgba(42, 27, 46, 0.55)",
  bubbleMine: "#6B2D5C",
  bubbleMineText: "#FFFFFF",
  bubbleTheirs: "#F3ECE6",
  skeleton: "#EBE2DA"
};

export const darkColors: ThemeColors = {
  background: "#171019",
  surface: "#241A27",
  surfaceMuted: "#2F2333",
  border: "#3B2E40",
  text: "#F7EFF5",
  textMuted: "#B5A6B8",
  primary: "#D78BC1",
  primaryText: "#241A27",
  primarySoft: "#3A2639",
  accent: "#FF8A6B",
  like: "#FF7A5C",
  pass: "#B5A6B8",
  danger: "#FF8A80",
  dangerSoft: "#3D1F22",
  success: "#6FD3A0",
  overlay: "rgba(0, 0, 0, 0.65)",
  bubbleMine: "#D78BC1",
  bubbleMineText: "#241A27",
  bubbleTheirs: "#2F2333",
  skeleton: "#33263A"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radii = { sm: 8, md: 14, lg: 22, pill: 999 } as const;
export const fontSizes = { xs: 12, sm: 14, md: 16, lg: 20, xl: 28, xxl: 36 } as const;
export const MIN_TOUCH = 44;
