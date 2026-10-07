import React from "react";
import { StyleSheet, View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { useTheme } from "../../theme/useTheme";
import type { ColorPalette } from "../../theme/palette";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type BadgeVariant = "default" | "primary" | "success" | "warning" | "danger" | "muted";
export type BadgeSize = "sm" | "md";

export type BadgeProps = Omit<ViewProps, "style"> & {
  label: string;
  size?: BadgeSize;
  style?: StyleProp<ViewStyle>;
  variant?: BadgeVariant;
};

const styles = StyleSheet.create({
  base: { borderRadius: radii.pill, borderWidth: 1, alignSelf: "flex-start", justifyContent: "center" },
  sizeSm: { minHeight: 24, paddingHorizontal: spacing.sm, paddingVertical: spacing.xxs },
  sizeMd: { minHeight: 28, paddingHorizontal: spacing.md, paddingVertical: spacing.xs }
});

function variantStyle(colors: ColorPalette, variant: BadgeVariant): ViewStyle {
  switch (variant) {
    case "primary":
      return { backgroundColor: colors.primarySoft, borderColor: colors.primaryBorder };
    case "success":
      return { backgroundColor: colors.successSoft, borderColor: colors.successBorder };
    case "warning":
      return { backgroundColor: colors.warningSoft, borderColor: colors.warningBorder };
    case "danger":
      return { backgroundColor: colors.dangerSoft, borderColor: colors.dangerBorder };
    case "muted":
      return { backgroundColor: colors.surfaceSubtle, borderColor: colors.border };
    default:
      return { backgroundColor: colors.backgroundElevated, borderColor: colors.border };
  }
}

const toneByVariant: Record<BadgeVariant, "danger" | "muted" | "primary" | "secondary" | "success"> = {
  default: "secondary",
  primary: "primary",
  success: "success",
  warning: "secondary",
  danger: "danger",
  muted: "muted"
};

export function Badge({ label, size = "md", style, variant = "default", ...props }: BadgeProps) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        styles.base,
        size === "sm" ? styles.sizeSm : styles.sizeMd,
        variantStyle(colors, variant),
        style
      ]}
      {...props}
    >
      <Text tone={toneByVariant[variant]} variant="caption" weight="bold">
        {label}
      </Text>
    </View>
  );
}
