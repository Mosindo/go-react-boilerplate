import { useMemo } from "react";
import { useTheme } from "../../theme";

export type Palette = {
  background: string;
  surface: string;
  surfaceMuted: string;
  text: string;
  textMuted: string;
  border: string;
  primary: string;
  primarySoft: string;
  onPrimary: string;
  danger: string;
  dangerSoft: string;
  success: string;
  overlay: string;
  like: string;
  nope: string;
  scrim: string;
  isDark: boolean;
};

/** Single place where this feature set reads theme colors (follows the system scheme). */
export function usePalette(): Palette {
  const { colors, isDark } = useTheme();
  return useMemo<Palette>(
    () => ({
      background: colors.background,
      surface: colors.surface,
      surfaceMuted: colors.surfaceSubtle,
      text: colors.text,
      textMuted: colors.textMuted,
      border: colors.border,
      primary: colors.primary,
      primarySoft: colors.primarySoft,
      onPrimary: colors.primaryForeground,
      danger: colors.danger,
      dangerSoft: colors.dangerSoft,
      success: colors.success,
      overlay: colors.overlay,
      like: isDark ? "#4CC9A0" : "#1E9E78",
      nope: isDark ? "#FF8A7A" : "#D9483B",
      scrim: colors.overlay,
      isDark
    }),
    [colors, isDark]
  );
}
