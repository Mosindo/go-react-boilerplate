import React from "react";
import { View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { radii, shadows, spacing } from "./tokens";

export type CardVariant = "default" | "muted" | "accent";
export type CardPadding = "sm" | "md" | "lg" | "xl";

export type CardProps = Omit<ViewProps, "style"> & {
  interactive?: boolean;
  selected?: boolean;
  variant?: CardVariant;
  padding?: CardPadding;
  style?: StyleProp<ViewStyle>;
};

const paddings: Record<CardPadding, number> = {
  sm: spacing.md,
  md: spacing.lg,
  lg: spacing.xl,
  xl: spacing.xxl
};

export function Card({
  children,
  interactive = false,
  padding = "md",
  selected = false,
  style,
  variant = "default",
  ...viewProps
}: CardProps) {
  const { colors } = useTheme();
  const variantStyle: ViewStyle | null =
    variant === "muted"
      ? { backgroundColor: colors.surfaceMuted }
      : variant === "accent"
        ? { backgroundColor: colors.surfaceAccent, borderColor: colors.primaryBorder }
        : null;

  return (
    <View
      style={[
        {
          borderRadius: radii.lg,
          borderWidth: 1,
          borderColor: colors.border,
          backgroundColor: colors.backgroundElevated,
          ...shadows.card
        },
        variantStyle,
        interactive ? { borderColor: colors.borderStrong } : null,
        selected ? { backgroundColor: colors.surfaceAccent, borderColor: colors.primary } : null,
        { padding: paddings[padding] },
        style
      ]}
      {...viewProps}
    >
      {children}
    </View>
  );
}
