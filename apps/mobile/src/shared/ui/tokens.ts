/**
 * Hearth design tokens — a warm, low-glare dark theme.
 * Text/background pairs meet WCAG AA contrast (≥ 4.5:1 for body text).
 */
export const colors = {
  background: "#14110F",
  backgroundElevated: "#1C1815",
  surface: "#211C19",
  surfaceRaised: "#2A2420",
  surfaceMuted: "#191513",
  border: "#3A322C",
  borderStrong: "#54483F",
  text: "#F6EFE9",
  textMuted: "#BCAEA2",
  textSubtle: "#8F8278",
  primary: "#EE8466",
  primaryPressed: "#D9694B",
  primarySoft: "rgba(238, 132, 102, 0.14)",
  primaryBorder: "rgba(238, 132, 102, 0.45)",
  onPrimary: "#1A0E09",
  accent: "#E9BD7B",
  accentSoft: "rgba(233, 189, 123, 0.14)",
  success: "#86CFA3",
  successSoft: "rgba(134, 207, 163, 0.14)",
  danger: "#F27A72",
  dangerSoft: "rgba(242, 122, 114, 0.14)",
  dangerBorder: "rgba(242, 122, 114, 0.45)",
  warning: "#E9BD7B",
  inverse: "#14110F",
  overlay: "rgba(8, 6, 5, 0.72)",
  scrim: "rgba(10, 8, 7, 0.0)",
  scrimStrong: "rgba(10, 8, 7, 0.88)"
} as const;

export const spacing = {
  xxs: 4,
  xs: 6,
  sm: 10,
  md: 14,
  lg: 18,
  xl: 24,
  xxl: 32,
  xxxl: 44
} as const;

export const radii = {
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 28,
  pill: 999
} as const;

export const fontWeights = {
  regular: "400",
  medium: "500",
  semibold: "600",
  bold: "700"
} as const;

export const typography = {
  display: { fontSize: 40, lineHeight: 46, letterSpacing: -1.2 },
  title: { fontSize: 30, lineHeight: 36, letterSpacing: -0.8 },
  heading: { fontSize: 21, lineHeight: 27, letterSpacing: -0.4 },
  body: { fontSize: 16, lineHeight: 23, letterSpacing: -0.1 },
  label: { fontSize: 14, lineHeight: 19, letterSpacing: 0 },
  caption: { fontSize: 12.5, lineHeight: 17, letterSpacing: 0.1 },
  eyebrow: { fontSize: 11.5, lineHeight: 15, letterSpacing: 1.4, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20, letterSpacing: -0.1 }
} as const;

export const controls = {
  button: { sm: 40, md: 50, lg: 56 },
  input: { md: 52, multiline: 120 },
  hitSlop: { top: 8, bottom: 8, left: 8, right: 8 }
} as const;

export const shadows = {
  card: {
    shadowColor: "#000000",
    shadowOpacity: 0.35,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 12 },
    elevation: 6
  },
  floating: {
    shadowColor: "#000000",
    shadowOpacity: 0.45,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 16 },
    elevation: 10
  }
} as const;

export const motion = {
  fast: 140,
  normal: 220,
  slow: 360
} as const;
