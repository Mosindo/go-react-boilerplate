import React from "react";
import { StyleSheet, View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { colors, radii, shadows, spacing } from "./tokens";

export type CardVariant = "default" | "muted" | "accent";
export type CardPadding = "sm" | "md" | "lg" | "xl";

export type CardProps = Omit<ViewProps, "style"> & {
  interactive?: boolean;
  selected?: boolean;
  variant?: CardVariant;
  padding?: CardPadding;
  style?: StyleProp<ViewStyle>;
};

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    ...shadows.card
  },
  muted: {
    backgroundColor: colors.surfaceMuted
  },
  accent: {
    backgroundColor: colors.surfaceAccent,
    borderColor: colors.primaryBorder
  },
  interactive: {
    borderColor: colors.borderStrong,
    ...shadows.floating
  },
  selected: {
    backgroundColor: colors.surfaceAccent,
    borderColor: colors.primaryBorder,
    ...shadows.floating
  },
  paddingSm: {
    padding: spacing.md
  },
  paddingMd: {
    padding: spacing.lg
  },
  paddingLg: {
    padding: spacing.xl
  },
  paddingXl: {
    padding: spacing.xxl
  }
});

const variantStyles: Record<CardVariant, ViewStyle | null> = {
  default: null,
  muted: styles.muted,
  accent: styles.accent
};

const paddingStyles: Record<CardPadding, ViewStyle> = {
  sm: styles.paddingSm,
  md: styles.paddingMd,
  lg: styles.paddingLg,
  xl: styles.paddingXl
};

export function Card({ children, padding = "md", style, variant = "default", ...props }: CardProps) {
  const { interactive = false, selected = false, ...viewProps } = props;
  return (
    <View
      style={[
        styles.base,
        variantStyles[variant],
        interactive ? styles.interactive : null,
        selected ? styles.selected : null,
        paddingStyles[padding],
        style
      ]}
      {...viewProps}
    >
      {children}
    </View>
  );
}
