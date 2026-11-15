import { Platform } from "react-native";

export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  surfaceRaised: string;
  border: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  accentSoft: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
};

// "Lueur" — a warm, low-light palette: ember accents on charcoal (dark) or
// on sand (light). Contrast of text on background is ≥ 7:1 in both themes.
export const darkPalette: Palette = {
  background: "#14100E",
  surface: "#1E1815",
  surfaceAlt: "#29211D",
  surfaceRaised: "#332925",
  border: "#3B302B",
  text: "#F7EFE9",
  textMuted: "#BFAEA3",
  textSubtle: "#8D7D73",
  primary: "#F2915F",
  primarySoft: "#3A2419",
  onPrimary: "#1C0F08",
  accent: "#E8708A",
  accentSoft: "#3A1C24",
  success: "#7BD3A0",
  danger: "#FF7A70",
  dangerSoft: "#3B1A17",
  overlay: "rgba(10, 7, 6, 0.72)",
  scrim: "rgba(0, 0, 0, 0.55)"
};

export const lightPalette: Palette = {
  background: "#FBF6F1",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EAE2",
  surfaceRaised: "#FFFFFF",
  border: "#E7DAD0",
  text: "#2A1D17",
  textMuted: "#6F5E54",
  textSubtle: "#9A887D",
  primary: "#C8582B",
  primarySoft: "#F8E3D7",
  onPrimary: "#FFFFFF",
  accent: "#B8445F",
  accentSoft: "#F6DCE2",
  success: "#2F8A57",
  danger: "#C73A30",
  dangerSoft: "#FBE3E0",
  overlay: "rgba(42, 29, 23, 0.55)",
  scrim: "rgba(0, 0, 0, 0.45)"
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

export const fonts = {
  display: Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" }),
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" })
} as const;

export const type = {
  display: { fontFamily: fonts.display, fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.5 },
  title: { fontFamily: fonts.display, fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.3 },
  heading: { fontFamily: fonts.body, fontSize: 18, lineHeight: 24, fontWeight: "700" as const },
  body: { fontFamily: fonts.body, fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  label: { fontFamily: fonts.body, fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontFamily: fonts.body, fontSize: 13, lineHeight: 18, fontWeight: "400" as const },
  micro: { fontFamily: fonts.body, fontSize: 11, lineHeight: 14, fontWeight: "700" as const, letterSpacing: 0.6 }
} as const;

export type TypeVariant = keyof typeof type;
