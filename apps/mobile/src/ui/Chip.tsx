import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radii, spacing, useColors } from "../theme";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
};

/** Selectable pill (interests, gender, distance presets). Read-only when onPress is omitted. */
export function Chip({ label, selected = false, onPress, disabled }: ChipProps) {
  const colors = useColors();
  const body = (
    <View
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primary : colors.surface,
          borderColor: selected ? colors.primary : colors.border
        }
      ]}
    >
      <Text style={{ color: selected ? colors.onPrimary : colors.text }} variant="small" weight="medium">
        {label}
      </Text>
    </View>
  );
  if (!onPress) return body;
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ selected, disabled: !!disabled }}
      disabled={disabled}
      onPress={onPress}
      style={[styles.hit, disabled && { opacity: 0.45 }]}
    >
      {body}
    </Pressable>
  );
}

export function Badge({ count }: { count: number }) {
  const colors = useColors();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={`${count} unread`} style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text style={{ color: colors.onPrimary }} variant="caption" weight="bold">
        {count > 99 ? "99+" : String(count)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  hit: { minHeight: MIN_TOUCH, justifyContent: "center" },
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    paddingHorizontal: 5,
    alignItems: "center",
    justifyContent: "center"
  }
});
