import React from "react";
import { StyleSheet, View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

export type BadgeVariant = "default" | "primary" | "success" | "warning" | "danger" | "muted";
export type BadgeSize = "sm" | "md";

export type BadgeProps = Omit<ViewProps, "style"> & {
  label: string;
  size?: BadgeSize;
  style?: StyleProp<ViewStyle>;
  variant?: BadgeVariant;
};

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    borderWidth: 1,
    alignSelf: "flex-start",
    justifyContent: "center"
  },
  sizeSm: {
    minHeight: 24,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xxs
  },
  sizeMd: {
    minHeight: 28,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  default: {
    backgroundColor: colors.backgroundElevated,
    borderColor: colors.border
  },
  primary: {
    backgroundColor: colors.primarySoft,
    borderColor: colors.primaryBorder
  },
  success: {
    backgroundColor: colors.successSoft,
    borderColor: colors.successBorder
  },
  warning: {
    backgroundColor: colors.warningSoft,
    borderColor: colors.warningBorder
  },
  danger: {
    backgroundColor: colors.dangerSoft,
    borderColor: colors.dangerBorder
  },
  muted: {
    backgroundColor: colors.surfaceSubtle,
    borderColor: colors.border
  }
});

const sizeStyles: Record<BadgeSize, ViewStyle> = {
  sm: styles.sizeSm,
  md: styles.sizeMd
};

const variantStyles: Record<BadgeVariant, ViewStyle> = {
  default: styles.default,
  primary: styles.primary,
  success: styles.success,
  warning: styles.warning,
  danger: styles.danger,
  muted: styles.muted
};

const toneByVariant: Record<BadgeVariant, "danger" | "muted" | "primary" | "secondary" | "success"> = {
  default: "secondary",
  primary: "primary",
  success: "success",
  warning: "secondary",
  danger: "danger",
  muted: "muted"
};

export function Badge({ label, size = "md", style, variant = "default", ...props }: BadgeProps) {
  return (
    <View style={[styles.base, sizeStyles[size], variantStyles[variant], style]} {...props}>
      <Text tone={toneByVariant[variant]} variant="caption" weight="bold">
        {label}
      </Text>
    </View>
  );
}
