import { Platform } from "react-native";

/**
 * Lueur identity: warm ivory and ink, an ember accent for desire and a
 * deep plum for intimacy. Calm surfaces, generous radii, one strong accent.
 */
export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  surfaceRaised: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  textInverse: string;
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

export const light: Palette = {
  background: "#FAF6F1",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE4",
  surfaceRaised: "#FFFFFF",
  border: "#E9E0D6",
  borderStrong: "#D4C7BA",
  text: "#231C1F",
  textMuted: "#6E6266",
  textSubtle: "#9A8E91",
  textInverse: "#FFFFFF",
  primary: "#C4513A",
  primaryPressed: "#A9412D",
  primarySoft: "#F7E3DC",
  onPrimary: "#FFFFFF",
  accent: "#5B2A4A",
  accentSoft: "#EFE3EA",
  success: "#2E7A58",
  successSoft: "#E1F1E8",
  danger: "#B3261E",
  dangerSoft: "#FBE4E2",
  overlay: "rgba(35, 28, 31, 0.45)",
  scrim: "rgba(20, 14, 17, 0.72)",
  shadow: "#2A1A20"
};

export const dark: Palette = {
  background: "#151113",
  surface: "#1F191C",
  surfaceMuted: "#2A2326",
  surfaceRaised: "#262023",
  border: "#362D31",
  borderStrong: "#4A3F43",
  text: "#F4EDE8",
  textMuted: "#B7AAAE",
  textSubtle: "#857A7E",
  textInverse: "#151113",
  primary: "#E2765C",
  primaryPressed: "#C9624A",
  primarySoft: "#3B2420",
  onPrimary: "#1B1012",
  accent: "#D3A9C4",
  accentSoft: "#33232D",
  success: "#6CC49A",
  successSoft: "#1C2E25",
  danger: "#F2867C",
  dangerSoft: "#3A1F1D",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.75)",
  shadow: "#000000"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 24, xxl: 32, xxxl: 48 } as const;
export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;

const serif = Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" });

export const type = {
  display: { fontFamily: serif, fontSize: 34, lineHeight: 40, letterSpacing: -0.5, fontWeight: "700" as const },
  title: { fontFamily: serif, fontSize: 26, lineHeight: 32, letterSpacing: -0.3, fontWeight: "700" as const },
  heading: { fontSize: 18, lineHeight: 24, letterSpacing: -0.2, fontWeight: "700" as const },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontSize: 13, lineHeight: 17, fontWeight: "400" as const },
  overline: { fontSize: 12, lineHeight: 16, letterSpacing: 1.2, fontWeight: "700" as const, textTransform: "uppercase" as const }
};

export type TypeVariant = keyof typeof type;
