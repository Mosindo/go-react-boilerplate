export const colors = {
  primary: "#4c6fff",
  primarySoft: "#e8edff",
  primaryBorder: "#cad5ff",
  primaryForeground: "#ffffff",
  secondary: "#475467",
  secondarySoft: "#eef2f6",
  background: "#f6f8fc",
  backgroundElevated: "#ffffff",
  surface: "#ffffff",
  surfaceMuted: "#f8fafc",
  surfaceSubtle: "#eef2f6",
  surfaceAccent: "#eef3ff",
  border: "#e3e8f2",
  borderStrong: "#c9d3e1",
  text: "#101828",
  textMuted: "#667085",
  textSubtle: "#98a2b3",
  success: "#027a48",
  successSoft: "#ecfdf3",
  successBorder: "#abefc6",
  warning: "#b54708",
  warningSoft: "#fff6e0",
  warningBorder: "#fedf89",
  danger: "#d92d20",
  dangerSoft: "#fef3f2",
  dangerBorder: "#fecdca",
  inverse: "#ffffff",
  overlay: "rgba(15, 23, 42, 0.38)"
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
    shadowColor: "#0b1220",
    shadowOpacity: 0.08,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 10 },
    elevation: 4
  },
  floating: {
    shadowColor: "#0b1220",
    shadowOpacity: 0.12,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 14 },
    elevation: 8
  },
  focus: {
    shadowColor: "#4c6fff",
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
