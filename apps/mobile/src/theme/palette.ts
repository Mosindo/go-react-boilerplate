// Seren visual identity: warm ivory + ink with a terracotta accent (light),
// deep aubergine night with a coral accent (dark).
export type Palette = {
  background: string;
  surface: string;
  surfaceRaised: string;
  surfaceSunken: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  border: string;
  borderStrong: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  accentSoft: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  shadow: string;
};

export const lightPalette: Palette = {
  background: "#FBF7F3",
  surface: "#FFFFFF",
  surfaceRaised: "#FFFFFF",
  surfaceSunken: "#F3ECE6",
  text: "#221C1F",
  textMuted: "#6B6064",
  textSubtle: "#9A8F93",
  border: "#ECE3DC",
  borderStrong: "#D9CCC3",
  primary: "#C4553B",
  primaryPressed: "#A94631",
  primarySoft: "#F7E4DD",
  onPrimary: "#FFFFFF",
  accent: "#5D3B5A",
  accentSoft: "#EFE6EE",
  success: "#2E7A57",
  successSoft: "#E3F2EA",
  danger: "#B42318",
  dangerSoft: "#FCE8E6",
  overlay: "rgba(34, 28, 31, 0.45)",
  scrim: "rgba(20, 14, 17, 0.62)",
  shadow: "#3B2A24"
};

export const darkPalette: Palette = {
  background: "#151114",
  surface: "#1E191C",
  surfaceRaised: "#272125",
  surfaceSunken: "#110E10",
  text: "#F5EFEB",
  textMuted: "#B3A8AC",
  textSubtle: "#7F7579",
  border: "#332B30",
  borderStrong: "#463C42",
  primary: "#E2775C",
  primaryPressed: "#C9644B",
  primarySoft: "#3B231E",
  onPrimary: "#1A1012",
  accent: "#C9A3C4",
  accentSoft: "#2F2330",
  success: "#5DBB8E",
  successSoft: "#1C2E25",
  danger: "#F2867B",
  dangerSoft: "#3A1D1A",
  overlay: "rgba(0, 0, 0, 0.55)",
  scrim: "rgba(0, 0, 0, 0.7)",
  shadow: "#000000"
};

export const spacing = {
  xxs: 4,
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 28,
  xxl: 40
} as const;

export const radii = {
  sm: 10,
  md: 16,
  lg: 24,
  xl: 32,
  pill: 999
} as const;

export const type = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700", letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.5 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "600", letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400", letterSpacing: 0 },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600", letterSpacing: 0.1 },
  caption: { fontSize: 13, lineHeight: 17, fontWeight: "400", letterSpacing: 0.1 },
  overline: { fontSize: 12, lineHeight: 16, fontWeight: "700", letterSpacing: 1.2 }
} as const;

export type TypeVariant = keyof typeof type;
