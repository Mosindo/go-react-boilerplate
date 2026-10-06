import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

export type BadgeTone = "neutral" | "primary" | "success" | "danger";

export type BadgeProps = {
  label: string;
  tone?: BadgeTone;
  selected?: boolean;
  onPress?: () => void;
  style?: StyleProp<ViewStyle>;
};

/** Small pill: a read-only tag, or a selectable chip when onPress is given. */
export function Badge({ label, tone = "neutral", selected = false, onPress, style }: BadgeProps) {
  const { colors } = useTheme();
  const base = {
    neutral: { bg: colors.surfaceMuted, fg: colors.textMuted, border: colors.border },
    primary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primaryBorder },
    success: { bg: colors.successSoft, fg: colors.success, border: colors.successBorder },
    danger: { bg: colors.dangerSoft, fg: colors.danger, border: colors.dangerBorder }
  }[tone];
  const active = selected ? { bg: colors.primary, fg: colors.primaryForeground, border: colors.primary } : base;

  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={onPress ? { selected } : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={[styles.root, { backgroundColor: active.bg, borderColor: active.border }, style]}
    >
      <Text style={{ color: active.fg }} variant="label" weight="semibold">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: {
    alignSelf: "flex-start",
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2
  }
});
