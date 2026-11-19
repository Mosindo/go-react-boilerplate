import { Platform } from "react-native";

export type ColorScheme = "light" | "dark";

export type Palette = {
  background: string;
  surface: string;
  surfaceRaised: string;
  surfaceMuted: string;
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
  like: string;
  pass: string;
  success: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  onPhoto: string;
};

// "Lueur" identity: deep wine and warm apricot on ivory (light) or ink plum (dark).
export const palettes: Record<ColorScheme, Palette> = {
  light: {
    background: "#FBF7F3",
    surface: "#FFFFFF",
    surfaceRaised: "#FFFFFF",
    surfaceMuted: "#F3ECE6",
    border: "#EADFD6",
    borderStrong: "#D8C8BC",
    text: "#231A22",
    textMuted: "#6D5F69",
    textSubtle: "#9A8C95",
    primary: "#8E2F4F",
    primaryPressed: "#76233F",
    primarySoft: "#F6E4EA",
    onPrimary: "#FFFFFF",
    accent: "#E58F4B",
    accentSoft: "#FCEBDD",
    like: "#8E2F4F",
    pass: "#6D5F69",
    success: "#2F7D5B",
    danger: "#C0392B",
    dangerSoft: "#FBE7E4",
    overlay: "rgba(35, 26, 34, 0.45)",
    scrim: "rgba(20, 12, 18, 0.62)",
    onPhoto: "#FFFFFF"
  },
  dark: {
    background: "#151015",
    surface: "#1F181F",
    surfaceRaised: "#2A2129",
    surfaceMuted: "#2A2129",
    border: "#382D36",
    borderStrong: "#4C3F49",
    text: "#F6EEE8",
    textMuted: "#BFAFB8",
    textSubtle: "#8E7F89",
    primary: "#E27A9A",
    primaryPressed: "#CC6585",
    primarySoft: "#3A2230",
    onPrimary: "#1A0F15",
    accent: "#F2A66A",
    accentSoft: "#3B2A1E",
    like: "#E27A9A",
    pass: "#BFAFB8",
    success: "#5CC491",
    danger: "#F07167",
    dangerSoft: "#3D1F1D",
    overlay: "rgba(0, 0, 0, 0.6)",
    scrim: "rgba(10, 6, 9, 0.7)",
    onPhoto: "#FFFFFF"
  }
};

export const spacing = { xxs: 4, xs: 8, sm: 12, md: 16, lg: 20, xl: 28, xxl: 40 } as const;

export const radii = { sm: 10, md: 16, lg: 24, xl: 32, pill: 999 } as const;

export const fonts = {
  display: Platform.select({ ios: "Georgia", android: "serif", default: "Georgia, 'Times New Roman', serif" }),
  body: Platform.select({ ios: "System", android: "sans-serif", default: "system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif" })
} as const;

export const type = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "700" as const, letterSpacing: -0.5 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700" as const, letterSpacing: -0.3 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: "700" as const },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" as const },
  label: { fontSize: 14, lineHeight: 19, fontWeight: "600" as const },
  caption: { fontSize: 12.5, lineHeight: 17, fontWeight: "500" as const }
} as const;

export type TypeVariant = keyof typeof type;

export const shadow = Platform.select({
  web: { boxShadow: "0 10px 30px rgba(35, 26, 34, 0.12)" },
  default: {
    shadowColor: "#231A22",
    shadowOpacity: 0.14,
    shadowRadius: 20,
    shadowOffset: { width: 0, height: 10 },
    elevation: 6
  }
}) as object;
