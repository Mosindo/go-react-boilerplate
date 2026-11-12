// Lueur design tokens. Warm ivory by day, warm charcoal by night, one ember accent.

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
  onPhoto: string;
  skeleton: string;
};

export const lightPalette: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F4EDE4",
  surfaceRaised: "#FFFFFF",
  border: "#EADFD3",
  borderStrong: "#D7C8B8",
  text: "#221A16",
  textMuted: "#6E625A",
  textSubtle: "#A0938A",
  primary: "#D2553A",
  primaryPressed: "#B44530",
  primarySoft: "#FBE4DC",
  onPrimary: "#FFFFFF",
  accent: "#4B2A4A",
  accentSoft: "#EFE3EE",
  success: "#2F7D5B",
  successSoft: "#E2F2EA",
  danger: "#B9382B",
  dangerSoft: "#FBE6E3",
  overlay: "rgba(34, 26, 22, 0.45)",
  scrim: "rgba(18, 12, 10, 0.62)",
  onPhoto: "#FFFFFF",
  skeleton: "#EFE6DC"
};

export const darkPalette: Palette = {
  background: "#14100E",
  surface: "#1E1815",
  surfaceMuted: "#29211D",
  surfaceRaised: "#251E1A",
  border: "#362C27",
  borderStrong: "#4A3E37",
  text: "#F6EEE7",
  textMuted: "#B9AA9E",
  textSubtle: "#7E7168",
  primary: "#EE7556",
  primaryPressed: "#D9603F",
  primarySoft: "#3B231C",
  onPrimary: "#1A0F0B",
  accent: "#D9B3D6",
  accentSoft: "#33222F",
  success: "#6CC59A",
  successSoft: "#1C2E25",
  danger: "#F08A7C",
  dangerSoft: "#3A1F1B",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.66)",
  onPhoto: "#FFFFFF",
  skeleton: "#2B231F"
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 24, xxl: 32, xxxl: 48 } as const;

export const radii = { sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;

export const type = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.5 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "600" as const, letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const, letterSpacing: 0 },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const, letterSpacing: 0 },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const, letterSpacing: 0.1 },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" as const, letterSpacing: 0.1 },
  overline: { fontSize: 12, lineHeight: 16, fontWeight: "700" as const, letterSpacing: 1.2 }
} as const;

export type TypeVariant = keyof typeof type;
