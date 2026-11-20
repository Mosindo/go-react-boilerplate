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
  primaryPressed: string;
  onPrimary: string;
  primarySoft: string;
  accent: string;
  accentSoft: string;
  danger: string;
  dangerSoft: string;
  success: string;
  successSoft: string;
  overlay: string;
  scrim: string;
  shadow: string;
};

// "Lueur" — a warm glow: ember accent, ivory text, deep aubergine night.
export const darkPalette: Palette = {
  background: "#14101A",
  surface: "#1D1724",
  surfaceAlt: "#272030",
  surfaceRaised: "#2F2739",
  border: "#3A3145",
  text: "#F7F0EA",
  textMuted: "#BCAFB9",
  textSubtle: "#8C7F8B",
  primary: "#F0825F",
  primaryPressed: "#DE6E4B",
  onPrimary: "#1B1016",
  primarySoft: "rgba(240, 130, 95, 0.16)",
  accent: "#E9BC7A",
  accentSoft: "rgba(233, 188, 122, 0.14)",
  danger: "#F26B77",
  dangerSoft: "rgba(242, 107, 119, 0.14)",
  success: "#62CBA5",
  successSoft: "rgba(98, 203, 165, 0.14)",
  overlay: "rgba(8, 5, 10, 0.72)",
  scrim: "rgba(10, 6, 12, 0.55)",
  shadow: "#000000"
};

export const lightPalette: Palette = {
  background: "#FBF6F1",
  surface: "#FFFFFF",
  surfaceAlt: "#F3EBE4",
  surfaceRaised: "#FFFFFF",
  border: "#E8DDD4",
  text: "#251B22",
  textMuted: "#6C5E67",
  textSubtle: "#978A92",
  primary: "#C4532F",
  primaryPressed: "#A94526",
  onPrimary: "#FFFFFF",
  primarySoft: "rgba(196, 83, 47, 0.10)",
  accent: "#A8752F",
  accentSoft: "rgba(168, 117, 47, 0.12)",
  danger: "#C8323F",
  dangerSoft: "rgba(200, 50, 63, 0.09)",
  success: "#1F8A63",
  successSoft: "rgba(31, 138, 99, 0.10)",
  overlay: "rgba(37, 27, 34, 0.55)",
  scrim: "rgba(20, 12, 18, 0.45)",
  shadow: "#3B2A33"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 24, xxl: 32, xxxl: 48 } as const;

export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;

const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const typography = {
  display: { fontFamily: serif, fontSize: 34, lineHeight: 40, letterSpacing: -0.5, fontWeight: "700" as const },
  title: { fontFamily: serif, fontSize: 26, lineHeight: 32, letterSpacing: -0.3, fontWeight: "700" as const },
  heading: { fontSize: 19, lineHeight: 25, letterSpacing: -0.2, fontWeight: "700" as const },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" as const },
  eyebrow: { fontSize: 12, lineHeight: 16, letterSpacing: 1.2, fontWeight: "700" as const, textTransform: "uppercase" as const }
};

export type TypographyVariant = keyof typeof typography;
