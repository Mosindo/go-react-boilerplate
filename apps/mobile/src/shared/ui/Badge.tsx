import React from "react";
import { View, type StyleProp, type ViewProps, type ViewStyle } from "react-native";
import { Text, type TextTone } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type BadgeVariant = "default" | "primary" | "success" | "warning" | "danger" | "muted";
export type BadgeSize = "sm" | "md";

export type BadgeProps = Omit<ViewProps, "style"> & {
  label: string;
  size?: BadgeSize;
  style?: StyleProp<ViewStyle>;
  variant?: BadgeVariant;
};

const makeStyles = (t: Theme) => ({
  base: {
    borderRadius: t.radii.pill,
    borderWidth: 1,
    alignSelf: "flex-start" as const,
    justifyContent: "center" as const
  },
  sm: { minHeight: 22, paddingHorizontal: t.spacing.sm, paddingVertical: 2 },
  md: { minHeight: 28, paddingHorizontal: t.spacing.md, paddingVertical: t.spacing.xxs },
  default: { backgroundColor: t.colors.surface, borderColor: t.colors.border },
  primary: { backgroundColor: t.colors.primarySoft, borderColor: t.colors.primary },
  success: { backgroundColor: t.colors.successSoft, borderColor: t.colors.success },
  warning: { backgroundColor: t.colors.warningSoft, borderColor: t.colors.warning },
  danger: { backgroundColor: t.colors.dangerSoft, borderColor: t.colors.danger },
  muted: { backgroundColor: t.colors.surfaceMuted, borderColor: t.colors.border }
});

const toneByVariant: Record<BadgeVariant, TextTone> = {
  default: "secondary",
  primary: "primary",
  success: "success",
  warning: "default",
  danger: "danger",
  muted: "muted"
};

export function Badge({ label, size = "md", style, variant = "default", ...props }: BadgeProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.base, styles[size], styles[variant], style]} {...props}>
      <Text tone={toneByVariant[variant]} variant="caption" weight="bold">
        {label}
      </Text>
    </View>
  );
}
