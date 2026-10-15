import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

export type BadgeTone = "neutral" | "brand" | "primary" | "success" | "warning" | "danger";

export type BadgeProps = {
  label: string | number;
  tone?: BadgeTone;
  style?: StyleProp<ViewStyle>;
};

export function Badge({ label, style, tone = "neutral" }: BadgeProps) {
  const palette = useTheme();
  const tones: Record<BadgeTone, { bg: string; fg: string }> = {
    neutral: { bg: palette.surfaceMuted, fg: palette.textMuted },
    brand: { bg: palette.brandSoft, fg: palette.brand },
    primary: { bg: palette.primarySoft, fg: palette.primary },
    success: { bg: palette.successSoft, fg: palette.success },
    warning: { bg: palette.warningSoft, fg: palette.warning },
    danger: { bg: palette.dangerSoft, fg: palette.danger }
  };
  const t = tones[tone];
  return (
    <View
      style={[
        {
          backgroundColor: t.bg,
          borderRadius: radii.pill,
          paddingHorizontal: spacing.md,
          paddingVertical: spacing.xxs + 1,
          alignSelf: "flex-start"
        },
        style
      ]}
    >
      <Text style={{ color: t.fg }} variant="caption" weight="semibold">
        {label}
      </Text>
    </View>
  );
}

/** Round counter used on tab icons and rows. Hidden when the count is zero. */
export function CountBadge({ count, style }: { count: number; style?: StyleProp<ViewStyle> }) {
  const palette = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View
      accessibilityLabel={`${count} non lu${count > 1 ? "s" : ""}`}
      style={[
        {
          minWidth: 20,
          height: 20,
          borderRadius: 10,
          paddingHorizontal: 5,
          backgroundColor: palette.primary,
          alignItems: "center",
          justifyContent: "center"
        },
        style
      ]}
    >
      <Text style={{ color: palette.onPrimary, fontSize: 11, lineHeight: 14 }} weight="bold">
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}
