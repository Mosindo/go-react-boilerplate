// Pure colour data, no imports: safe to import from anywhere (including tests).

export const lightColors = {
  primary: "#C8472F",
  primarySoft: "#FCEAE4",
  primaryBorder: "#F2C5B8",
  primaryForeground: "#FFFFFF",
  secondary: "#5A4D44",
  secondarySoft: "#F1EAE2",
  background: "#FBF7F2",
  backgroundElevated: "#FFFFFF",
  surface: "#FFFFFF",
  surfaceMuted: "#F6F0E9",
  surfaceSubtle: "#EFE7DE",
  surfaceAccent: "#FDF1EC",
  border: "#E9DFD4",
  borderStrong: "#D3C5B6",
  text: "#2A211C",
  textMuted: "#6E6158",
  textSubtle: "#9B8E84",
  success: "#2F7D4F",
  successSoft: "#EAF6EE",
  successBorder: "#B5DEC3",
  warning: "#A5560D",
  warningSoft: "#FFF4E0",
  warningBorder: "#F5D79B",
  danger: "#C23A2B",
  dangerSoft: "#FDEEEC",
  dangerBorder: "#F3C0BA",
  /** Text colour to use on top of `primary` / `text` coloured backgrounds. */
  inverse: "#FFFFFF",
  /** Always white: text/icons drawn over photos. */
  onImage: "#FFFFFF",
  overlay: "rgba(32, 20, 14, 0.42)"
} as const;

export type ColorPalette = { [K in keyof typeof lightColors]: string };

export const darkColors: ColorPalette = {
  primary: "#F07A60",
  primarySoft: "#3B211A",
  primaryBorder: "#6B3426",
  primaryForeground: "#1E100B",
  secondary: "#CDBFB4",
  secondarySoft: "#2A221E",
  background: "#15100E",
  backgroundElevated: "#201A17",
  surface: "#201A17",
  surfaceMuted: "#1B1512",
  surfaceSubtle: "#2A221E",
  surfaceAccent: "#2D1D18",
  border: "#352C27",
  borderStrong: "#4D4038",
  text: "#F6EEE8",
  textMuted: "#B5A79D",
  textSubtle: "#87796F",
  success: "#6CCB8E",
  successSoft: "#15271C",
  successBorder: "#27533A",
  warning: "#F0B25A",
  warningSoft: "#2E2210",
  warningBorder: "#5E4519",
  danger: "#FF8A7A",
  dangerSoft: "#32180F",
  dangerBorder: "#6B2E26",
  inverse: "#1E100B",
  onImage: "#FFFFFF",
  overlay: "rgba(0, 0, 0, 0.6)"
};

export type ColorScheme = "light" | "dark";

export function paletteFor(scheme: ColorScheme | null | undefined): ColorPalette {
  return scheme === "dark" ? darkColors : lightColors;
}
