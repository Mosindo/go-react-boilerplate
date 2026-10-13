import type { TextStyle, ViewStyle } from "react-native";

export type Colors = {
  primary: string;
  primarySoft: string;
  primaryForeground: string;
  accent: string;
  background: string;
  surface: string;
  surfaceMuted: string;
  border: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  success: string;
  successSoft: string;
  warning: string;
  warningSoft: string;
  danger: string;
  dangerSoft: string;
  like: string;
  pass: string;
  overlay: string;
  bubbleMine: string;
  bubbleMineText: string;
  bubbleTheirs: string;
};

export const lightColors: Colors = {
  primary: "#D8502F",
  primarySoft: "#FCE9E2",
  primaryForeground: "#FFFFFF",
  accent: "#5B2A4E",
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceMuted: "#F3ECE5",
  border: "#E7DDD3",
  text: "#2B1F1B",
  textMuted: "#6F5F57",
  textSubtle: "#9A8B82",
  success: "#2F7D4F",
  successSoft: "#E4F3EA",
  warning: "#A35A0B",
  warningSoft: "#FDF0DB",
  danger: "#C0322B",
  dangerSoft: "#FBE7E5",
  like: "#D8502F",
  pass: "#6F5F57",
  overlay: "rgba(24, 14, 10, 0.45)",
  bubbleMine: "#D8502F",
  bubbleMineText: "#FFFFFF",
  bubbleTheirs: "#F0E7DF"
};

export const darkColors: Colors = {
  primary: "#F0764F",
  primarySoft: "#3A211A",
  primaryForeground: "#1A0F0B",
  accent: "#D9A3C9",
  background: "#161110",
  surface: "#211A18",
  surfaceMuted: "#2B2220",
  border: "#3A2F2B",
  text: "#F6EDE7",
  textMuted: "#BBAAA1",
  textSubtle: "#8C7C73",
  success: "#6CC28D",
  successSoft: "#1B2E23",
  warning: "#E5A04A",
  warningSoft: "#33270F",
  danger: "#F27E76",
  dangerSoft: "#3A1C1A",
  like: "#F0764F",
  pass: "#BBAAA1",
  overlay: "rgba(0, 0, 0, 0.6)",
  bubbleMine: "#F0764F",
  bubbleMineText: "#1A0F0B",
  bubbleTheirs: "#2E2522"
};

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
  sm: 12,
  md: 16,
  lg: 22,
  xl: 30,
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
  body: { fontSize: 16, lineHeight: 23 },
  label: { fontSize: 13, lineHeight: 18, letterSpacing: 0.2 },
  title: { fontSize: fontSizes.xxl, lineHeight: 42, letterSpacing: -1 },
  heading: { fontSize: 22, lineHeight: 28, letterSpacing: -0.4 },
  caption: { fontSize: fontSizes.xs, lineHeight: 16 },
  eyebrow: { fontSize: fontSizes.xxs, lineHeight: 16, letterSpacing: 1.4, textTransform: "uppercase" as const },
  button: { fontSize: 16, lineHeight: 20, letterSpacing: -0.1 }
} as const satisfies Record<string, TextStyle>;

export const controls = {
  button: { sm: 40, md: 50, lg: 56 },
  input: { md: 52, multiline: 120 },
  avatar: { sm: 40, md: 52, lg: 72 }
} as const;

export function cardShadow(scheme: "light" | "dark"): ViewStyle {
  return scheme === "dark"
    ? { elevation: 0 }
    : {
        shadowColor: "#3b1f12",
        shadowOpacity: 0.1,
        shadowRadius: 22,
        shadowOffset: { width: 0, height: 10 },
        elevation: 4
      };
}

export type TypographyToken = keyof typeof typography;
