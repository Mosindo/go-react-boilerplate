// Design tokens for "Lien": warm, adult, minimal. Light and dark palettes share the same keys.
export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  primary: string;
  primaryPressed: string;
  primarySoft: string;
  onPrimary: string;
  accent: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  warning: string;
  overlay: string;
  scrim: string;
  like: string;
  pass: string;
};

export const lightPalette: Palette = {
  background: "#FBF7F4",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE7",
  border: "#E8DDD6",
  text: "#2B1B24",
  textMuted: "#6E5D66",
  textSubtle: "#8F7F88",
  primary: "#B93A52",
  primaryPressed: "#9C2E44",
  primarySoft: "#F8E4E8",
  onPrimary: "#FFFFFF",
  accent: "#3B1F3A",
  success: "#237A57",
  successSoft: "#E1F3EA",
  danger: "#C0392B",
  dangerSoft: "#FBE7E4",
  warning: "#9A5B00",
  overlay: "rgba(35, 20, 30, 0.45)",
  scrim: "rgba(20, 8, 16, 0.55)",
  like: "#2E9E6B",
  pass: "#8F7F88"
};

export const darkPalette: Palette = {
  background: "#160F18",
  surface: "#211826",
  surfaceMuted: "#2B2030",
  border: "#3A2D40",
  text: "#F7EEF1",
  textMuted: "#BBA9B3",
  textSubtle: "#9A8892",
  primary: "#E4657D",
  primaryPressed: "#EE8397",
  primarySoft: "#432431",
  onPrimary: "#1B0E14",
  accent: "#F3C9D3",
  success: "#52C597",
  successSoft: "#173126",
  danger: "#F0766A",
  dangerSoft: "#3D1E1B",
  warning: "#E5A33C",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "rgba(0, 0, 0, 0.6)",
  like: "#52C597",
  pass: "#9A8892"
};

export const spacing = { xxs: 4, xs: 6, sm: 10, md: 14, lg: 18, xl: 24, xxl: 32, xxxl: 40 } as const;
export const radii = { xs: 8, sm: 10, md: 14, lg: 20, xl: 28, pill: 999 } as const;
export const fontSizes = { xxs: 11, xs: 12, sm: 14, md: 16, lg: 20, xl: 28, xxl: 36 } as const;
export const fontWeights = { regular: "400", medium: "500", semibold: "600", bold: "700" } as const;

export const typography = {
  body: { fontSize: 16, lineHeight: 24 },
  bodySmall: { fontSize: 14, lineHeight: 20 },
  label: { fontSize: 13, lineHeight: 18, letterSpacing: 0.2 },
  title: { fontSize: 32, lineHeight: 38, letterSpacing: -0.6 },
  heading: { fontSize: 22, lineHeight: 28, letterSpacing: -0.3 },
  caption: { fontSize: 12, lineHeight: 16 },
  eyebrow: { fontSize: 11, lineHeight: 16, letterSpacing: 1.3, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20 }
} as const;

export const controls = {
  button: { sm: 40, md: 50, lg: 56 },
  input: { md: 52, multiline: 120 },
  avatar: { sm: 40, md: 48, lg: 64 }
} as const;

export const shadows = {
  card: { shadowColor: "#2B1B24", shadowOpacity: 0.1, shadowRadius: 18, shadowOffset: { width: 0, height: 8 }, elevation: 4 },
  floating: { shadowColor: "#2B1B24", shadowOpacity: 0.18, shadowRadius: 26, shadowOffset: { width: 0, height: 12 }, elevation: 10 }
} as const;
