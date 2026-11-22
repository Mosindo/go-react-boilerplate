/**
 * Lueur design tokens. Dark-first, warm and quiet: an ink/aubergine base,
 * an apricot accent for intent, sage for positive actions.
 */
export const colors = {
  background: "#14111A",
  surface: "#1D1925",
  surfaceRaised: "#262130",
  surfaceHigh: "#302A3C",
  border: "#3A3347",
  borderStrong: "#4D4560",

  text: "#F6F1EA",
  textMuted: "#B3AABF",
  textSubtle: "#857C93",
  textOnAccent: "#1B1420",

  accent: "#F0A979",
  accentStrong: "#F6BE96",
  accentSoft: "rgba(240, 169, 121, 0.14)",

  like: "#86D4B2",
  likeSoft: "rgba(134, 212, 178, 0.16)",
  pass: "#C9C1D6",
  danger: "#F27D7D",
  dangerSoft: "rgba(242, 125, 125, 0.14)",
  warning: "#F3C969",

  overlay: "rgba(10, 8, 14, 0.72)",
  scrimTop: "rgba(20, 17, 26, 0)",
  scrimBottom: "rgba(20, 17, 26, 0.92)"
} as const;

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

export const typography = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700", letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.5 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "600", letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400", letterSpacing: 0 },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600", letterSpacing: 0.1 },
  caption: { fontSize: 13, lineHeight: 17, fontWeight: "400", letterSpacing: 0.1 },
  eyebrow: { fontSize: 11, lineHeight: 14, fontWeight: "700", letterSpacing: 1.4 }
} as const;

export const shadows = {
  card: {
    shadowColor: "#000000",
    shadowOpacity: 0.35,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 12 },
    elevation: 8
  }
} as const;

export const layout = {
  maxContentWidth: 560,
  hitSlop: { top: 10, bottom: 10, left: 10, right: 10 }
} as const;
