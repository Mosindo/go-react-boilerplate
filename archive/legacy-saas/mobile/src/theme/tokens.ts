import { createTokens } from "tamagui";
import { colors, fontSizes, radii, spacing } from "../shared/ui";

export const tokens = createTokens({
  color: {
    background: colors.background,
    backgroundElevated: colors.backgroundElevated,
    text: colors.text,
    textMuted: colors.textMuted,
    border: colors.border,
    primary: colors.primary,
    secondary: colors.secondary,
    success: colors.success,
    warning: colors.warning,
    danger: colors.danger,
    muted: colors.surfaceMuted,
    accent: colors.surfaceAccent
  },
  space: {
    0: 0,
    0.5: spacing.xxs,
    1: spacing.xs,
    2: spacing.sm,
    3: spacing.md,
    4: spacing.lg,
    5: spacing.xl,
    6: spacing.xxl,
    7: spacing.xxxl,
    true: spacing.lg
  },
  size: {
    0: 0,
    0.5: fontSizes.xxs,
    1: fontSizes.xs,
    2: fontSizes.sm,
    3: fontSizes.md,
    4: fontSizes.lg,
    5: fontSizes.xl,
    6: fontSizes.xxl,
    true: fontSizes.md
  },
  radius: {
    0: 0,
    0.5: radii.xs,
    1: radii.sm,
    2: radii.md,
    3: radii.lg,
    4: radii.xl,
    true: radii.md
  },
  zIndex: {
    0: 0,
    1: 100,
    2: 200,
    3: 300,
    4: 400,
    5: 500
  }
});
