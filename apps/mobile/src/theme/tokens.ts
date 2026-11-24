// Lueur design tokens — a warm, low-light palette: ink backgrounds, ember
// accent, cream text. Contrast ratios of text on backgrounds exceed WCAG AA.
export const colors = {
  background: "#141014",
  surface: "#1D171C",
  surfaceRaised: "#272026",
  surfacePressed: "#30282F",
  border: "#3A3138",
  borderStrong: "#54484F",
  text: "#F7EFEA",
  textMuted: "#BFB0B6",
  textSubtle: "#8C7D84",
  primary: "#FF7A5C",
  primaryPressed: "#F0654A",
  primarySoft: "rgba(255, 122, 92, 0.14)",
  onPrimary: "#1B0E0A",
  accent: "#E9B872",
  accentSoft: "rgba(233, 184, 114, 0.16)",
  success: "#7FD4A9",
  successSoft: "rgba(127, 212, 169, 0.14)",
  danger: "#FF6F6F",
  dangerSoft: "rgba(255, 111, 111, 0.14)",
  overlay: "rgba(8, 5, 8, 0.72)",
  scrim: "rgba(12, 8, 11, 0.55)",
  white: "#FFFFFF"
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

export const type = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.5 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "600" as const, letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" as const },
  overline: { fontSize: 12, lineHeight: 16, fontWeight: "700" as const, letterSpacing: 1.2, textTransform: "uppercase" as const }
} as const;

export type TypeVariant = keyof typeof type;

export const shadow = {
  card: {
    shadowColor: "#000",
    shadowOpacity: 0.35,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 10 },
    elevation: 8
  }
} as const;

export const layout = {
  maxContentWidth: 560,
  hitSlop: { top: 10, bottom: 10, left: 10, right: 10 }
} as const;
