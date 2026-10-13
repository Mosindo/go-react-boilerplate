import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type BadgeProps = {
  count?: number;
  label?: string;
  tone?: "primary" | "neutral" | "success";
  style?: StyleProp<ViewStyle>;
};

/** Unread counter (count) or small status pill (label). Renders nothing for a zero count. */
export function Badge({ count, label, tone = "primary", style }: BadgeProps) {
  const { colors } = useTheme();
  if (count !== undefined && count <= 0) {
    return null;
  }
  const bg = tone === "primary" ? colors.primary : tone === "success" ? colors.success : colors.surfaceMuted;
  const fg = tone === "neutral" ? colors.textMuted : tone === "primary" ? colors.primaryForeground : "#FFFFFF";
  const text = label ?? (count !== undefined && count > 99 ? "99+" : String(count));
  return (
    <View accessibilityLabel={label ?? `${count} non lus`} style={[styles.badge, { backgroundColor: bg }, style]}>
      <Text style={{ color: fg }} variant="caption" weight="bold">
        {text}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    minWidth: 22,
    height: 22,
    paddingHorizontal: spacing.xs + 2,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center"
  }
});
