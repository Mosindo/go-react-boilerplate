/**
 * Lumen design tokens: warm terracotta/coral accent on soft cream, with a
 * matching dark palette. Text/background pairs meet WCAG AA (4.5:1).
 */
export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  bubbleMine: string;
  bubbleMineText: string;
  bubbleTheirs: string;
  bubbleTheirsText: string;
  skeleton: string;
};

export const lightPalette: Palette = {
  background: "#FBF6F0",
  surface: "#FFFFFF",
  surfaceMuted: "#F3EBE2",
  border: "#E6DACD",
  text: "#2B211C",
  textMuted: "#6B5B51",
  primary: "#B8472F",
  primaryPressed: "#9A3A25",
  primarySoft: "#F8E3DB",
  onPrimary: "#FFFFFF",
  success: "#2F7A55",
  successSoft: "#DFF1E7",
  danger: "#B3261E",
  dangerSoft: "#FADFDC",
  overlay: "rgba(43, 33, 28, 0.5)",
  bubbleMine: "#B8472F",
  bubbleMineText: "#FFFFFF",
  bubbleTheirs: "#F0E6DB",
  bubbleTheirsText: "#2B211C",
  skeleton: "#EADFD3"
};

export const darkPalette: Palette = {
  background: "#17110E",
  surface: "#241B17",
  surfaceMuted: "#2F241E",
  border: "#42342C",
  text: "#F6EDE6",
  textMuted: "#BCA99C",
  primary: "#E0795F",
  primaryPressed: "#EC8D75",
  primarySoft: "#432720",
  onPrimary: "#1B0F0B",
  success: "#6CC49A",
  successSoft: "#1E3A2D",
  danger: "#F2827A",
  dangerSoft: "#4A2220",
  overlay: "rgba(0, 0, 0, 0.62)",
  bubbleMine: "#E0795F",
  bubbleMineText: "#1B0F0B",
  bubbleTheirs: "#35291F",
  bubbleTheirsText: "#F6EDE6",
  skeleton: "#33271F"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 24, xl: 32, xxl: 48 } as const;

export const radii = { sm: 8, md: 14, lg: 20, xl: 28, pill: 999 } as const;

export const fontSizes = { caption: 12, small: 14, body: 16, title: 20, heading: 26, display: 34 } as const;

export const lineHeights = { caption: 16, small: 20, body: 23, title: 26, heading: 32, display: 40 } as const;

export const fontWeights = { regular: "400", medium: "500", semibold: "600", bold: "700" } as const;

/** Minimum touch target (iOS HIG / Material guidance). */
export const MIN_TOUCH = 44;
