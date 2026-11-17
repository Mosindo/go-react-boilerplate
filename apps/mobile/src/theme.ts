// Lueur visual identity: a warm, low-light palette ("candlelight") designed for
// comfortable evening use. Contrast ratios of text tokens on `bg` are >= 4.5:1.
export const colors = {
  bg: "#15101A",
  surface: "#1F1824",
  surfaceRaised: "#2A2130",
  surfacePressed: "#33293A",
  border: "#3A2F40",
  borderStrong: "#55475C",
  text: "#F6EFEA",
  textMuted: "#C2B5BC",
  textSubtle: "#988A92",
  accent: "#FF7A59",
  accentPressed: "#E8664A",
  accentSoft: "rgba(255, 122, 89, 0.14)",
  accentText: "#1B0F0B",
  gold: "#F2B866",
  goldSoft: "rgba(242, 184, 102, 0.14)",
  pass: "#A99BB0",
  success: "#7BD3A0",
  successSoft: "rgba(123, 211, 160, 0.14)",
  danger: "#FF6B7A",
  dangerSoft: "rgba(255, 107, 122, 0.14)",
  overlay: "rgba(10, 6, 12, 0.72)",
  scrim: "rgba(10, 6, 12, 0.55)",
  white: "#FFFFFF"
} as const;

export const spacing = {
  xxs: 2,
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  xxxl: 48
} as const;

export const radii = {
  sm: 8,
  md: 12,
  lg: 18,
  xl: 26,
  pill: 999
} as const;

export const typography = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.5 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "600" as const, letterSpacing: -0.2 },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  bodyStrong: { fontSize: 16, lineHeight: 23, fontWeight: "600" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "500" as const },
  caption: { fontSize: 12.5, lineHeight: 17, fontWeight: "400" as const },
  overline: { fontSize: 11.5, lineHeight: 15, fontWeight: "700" as const, letterSpacing: 1.2, textTransform: "uppercase" as const }
} as const;

export type TypographyVariant = keyof typeof typography;

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
  tabBarHeight: 64
} as const;
