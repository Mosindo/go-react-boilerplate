// Warm, adult palette: cream surfaces, ink text, raspberry accent.
export const colors = {
  primary: "#B8435A",
  primarySoft: "#F9E8EC",
  primaryBorder: "#EBC3CD",
  primaryForeground: "#ffffff",
  secondary: "#5E4F55",
  secondarySoft: "#F1E9E4",
  background: "#FAF6F1",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#F4EDE5",
  surfaceSubtle: "#EDE3D9",
  surfaceAccent: "#FBEFF1",
  border: "#E8DDD2",
  borderStrong: "#D5C6B8",
  text: "#2A2124",
  textMuted: "#6E6165",
  textSubtle: "#A09397",
  success: "#2F7D5B",
  successSoft: "#E9F5EE",
  successBorder: "#B5DCC7",
  warning: "#A5661A",
  warningSoft: "#FFF4E0",
  warningBorder: "#F0D49B",
  danger: "#C0392B",
  dangerSoft: "#FDEEEC",
  dangerBorder: "#F2C4BE",
  like: "#2F9E6E",
  pass: "#8A7B80",
  inverse: "#ffffff",
  overlay: "rgba(42, 33, 36, 0.45)",
  scrim: "rgba(20, 12, 14, 0.62)"
} as const;

export const spacing = {
  xxs: 4,
  xs: 6,
  sm: 10,
  md: 14,
  lg: 18,
  xl: 24,
  xxl: 32,
  xxxl: 40
} as const;

export const radii = {
  xs: 8,
  sm: 10,
  md: 14,
  lg: 20,
  xl: 28,
  pill: 999
} as const;

export const fontSizes = {
  xxs: 11,
  xs: 12,
  sm: 14,
  md: 16,
  lg: 20,
  xl: 28,
  xxl: 36
} as const;

export const fontWeights = {
  regular: "400",
  medium: "500",
  semibold: "600",
  bold: "700"
} as const;

export const typography = {
  body: {
    fontSize: 15,
    lineHeight: 24,
    letterSpacing: -0.1
  },
  label: {
    fontSize: 13,
    lineHeight: 18,
    letterSpacing: 0.1
  },
  title: {
    fontSize: fontSizes.xxl,
    lineHeight: 40,
    letterSpacing: -1
  },
  heading: {
    fontSize: 22,
    lineHeight: 28,
    letterSpacing: -0.5
  },
  caption: {
    fontSize: fontSizes.xs,
    lineHeight: 16,
    letterSpacing: 0
  },
  eyebrow: {
    fontSize: fontSizes.xxs,
    lineHeight: 16,
    letterSpacing: 1.3,
    textTransform: "uppercase" as const
  },
  button: {
    fontSize: 15,
    lineHeight: 20,
    letterSpacing: -0.2
  }
} as const;

export const controls = {
  button: {
    sm: 40,
    md: 50,
    lg: 56
  },
  input: {
    md: 52,
    multiline: 128
  },
  avatar: {
    sm: 40,
    md: 44,
    lg: 56
  }
} as const;

export const shadows = {
  card: {
    shadowColor: "#2A2124",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#2A2124",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#B8435A",
    shadowOpacity: 0.16,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 0 },
    elevation: 0
  }
} as const;

export const ui = {
  colors,
  spacing,
  radii,
  fontSizes,
  fontWeights,
  typography,
  controls,
  shadows
} as const;

export type ColorToken = keyof typeof colors;
export type TypographyToken = keyof typeof typography;
