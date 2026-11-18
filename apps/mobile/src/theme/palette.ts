// Lueur visual identity: warm, adult, quiet. Terracotta accent on cream paper
// in light mode, on deep espresso in dark mode. "scrim" is the opaque backdrop
// of full-screen moments such as a new match.
export type Palette = {
  background: string;
  surface: string;
  surfaceAlt: string;
  surfaceRaised: string;
  border: string;
  borderStrong: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  textOnAccent: string;
  accent: string;
  accentPressed: string;
  accentSoft: string;
  gold: string;
  success: string;
  successSoft: string;
  danger: string;
  dangerSoft: string;
  overlay: string;
  scrim: string;
  shadow: string;
};

export const lightPalette: Palette = {
  background: "#FBF7F2",
  surface: "#FFFFFF",
  surfaceAlt: "#F3ECE4",
  surfaceRaised: "#FFFFFF",
  border: "#E9DFD4",
  borderStrong: "#D4C6B8",
  text: "#1F1A17",
  textMuted: "#6B5F57",
  textSubtle: "#988B81",
  textOnAccent: "#FFFFFF",
  accent: "#B9503A",
  accentPressed: "#9C412E",
  accentSoft: "#F5E2DB",
  gold: "#C9952E",
  success: "#2F7351",
  successSoft: "#E3F1E8",
  danger: "#B3261E",
  dangerSoft: "#FBE6E4",
  overlay: "rgba(24, 18, 14, 0.55)",
  scrim: "#1B1411",
  shadow: "#2B1D14"
};

export const darkPalette: Palette = {
  background: "#14110F",
  surface: "#1D1916",
  surfaceAlt: "#28221E",
  surfaceRaised: "#25201C",
  border: "#362E28",
  borderStrong: "#4A4038",
  text: "#F4EEE7",
  textMuted: "#BDB0A5",
  textSubtle: "#8A7E74",
  textOnAccent: "#FFFFFF",
  accent: "#E07A5F",
  accentPressed: "#C9654B",
  accentSoft: "#3B2620",
  gold: "#E2B75A",
  success: "#6FB58D",
  successSoft: "#1E3027",
  danger: "#F2867B",
  dangerSoft: "#3A1F1C",
  overlay: "rgba(0, 0, 0, 0.6)",
  scrim: "#0E0B09",
  shadow: "#000000"
};
