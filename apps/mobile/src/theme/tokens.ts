// "Lueur" design tokens. Colours come in a light and a dark scheme; contrast of text on its
// background is >= 4.5:1 for body text in both (checked in theme.test.ts).
export type Colors = {
  background: string;
  surface: string;
  surfaceMuted: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  onPrimary: string;
  primarySoft: string;
  accent: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  onImage: string;
  bubbleMine: string;
  onBubbleMine: string;
  bubbleTheirs: string;
};

export const lightColors: Colors = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  text: "#231A24",
  textMuted: "#6B5F6E",
  border: "#E7DDD3",
  primary: "#B8381F",
  onPrimary: "#FFFFFF",
  primarySoft: "#FBE3DC",
  accent: "#3A1F3D",
  success: "#23724F",
  danger: "#B3261E",
  dangerSoft: "#FBE4E2",
  overlay: "rgba(35, 26, 36, 0.55)",
  onImage: "#FFFFFF",
  bubbleMine: "#3A1F3D",
  onBubbleMine: "#FFFFFF",
  bubbleTheirs: "#F0E7DE"
};

export const darkColors: Colors = {
  background: "#150F18",
  surface: "#21182A",
  surfaceMuted: "#2B2036",
  text: "#F6EFF3",
  textMuted: "#B8AABD",
  border: "#3A2D46",
  primary: "#FF7A5C",
  onPrimary: "#2A0E08",
  primarySoft: "#43211F",
  accent: "#E9D5F0",
  success: "#6FD1A2",
  danger: "#FF8A80",
  dangerSoft: "#43211F",
  overlay: "rgba(0, 0, 0, 0.6)",
  onImage: "#FFFFFF",
  bubbleMine: "#FF7A5C",
  onBubbleMine: "#2A0E08",
  bubbleTheirs: "#2B2036"
};

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32, xxxl: 48 } as const;
export const radii = { sm: 8, md: 14, lg: 20, xl: 28, pill: 999 } as const;
export const fontSizes = { xs: 12, sm: 14, md: 16, lg: 18, xl: 22, xxl: 28, hero: 38 } as const;
export const layout = { maxContentWidth: 560 } as const;
