export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  onPrimary: string;
  accent: string;
  onAccent: string;
  highlight: string;
  danger: string;
  onDanger: string;
  success: string;
  overlay: string;
  skeleton: string;
  like: string;
  pass: string;
  bubbleMine: string;
  onBubbleMine: string;
  bubbleTheirs: string;
  onBubbleTheirs: string;
};

/** Deep plum with terracotta and amber accents on a warm cream. */
export const lightPalette: Palette = {
  background: "#FBF5EE",
  surface: "#FFFFFF",
  surfaceAlt: "#F3E8DC",
  border: "#E6D6C6",
  text: "#2B1228",
  textMuted: "#6E5568",
  primary: "#5B2150",
  onPrimary: "#FFFFFF",
  accent: "#B8472E",
  onAccent: "#FFFFFF",
  highlight: "#E9A23B",
  danger: "#B3261E",
  onDanger: "#FFFFFF",
  success: "#2E7D4F",
  overlay: "rgba(43, 18, 40, 0.55)",
  skeleton: "#EADBCB",
  like: "#B8472E",
  pass: "#6E5568",
  bubbleMine: "#5B2150",
  onBubbleMine: "#FFFFFF",
  bubbleTheirs: "#F3E8DC",
  onBubbleTheirs: "#2B1228"
};

export const darkPalette: Palette = {
  background: "#1B0F1A",
  surface: "#281725",
  surfaceAlt: "#35202F",
  border: "#4A3144",
  text: "#F6EBE1",
  textMuted: "#BFA8B6",
  primary: "#F0B45E",
  onPrimary: "#2B1228",
  accent: "#E5826A",
  onAccent: "#2B1228",
  highlight: "#F0B45E",
  danger: "#FF8A80",
  onDanger: "#2B1228",
  success: "#7BD39B",
  overlay: "rgba(0, 0, 0, 0.65)",
  skeleton: "#3B2536",
  like: "#E5826A",
  pass: "#BFA8B6",
  bubbleMine: "#F0B45E",
  onBubbleMine: "#2B1228",
  bubbleTheirs: "#35202F",
  onBubbleTheirs: "#F6EBE1"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32 } as const;
export const radius = { sm: 10, md: 16, lg: 24, pill: 999 } as const;
export const MIN_TOUCH = 44;
export const CONTENT_MAX_WIDTH = 560;
