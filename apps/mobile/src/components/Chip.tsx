import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { colors, radii, spacing, Text } from "../shared/ui";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
};

export function Chip({ label, selected = false, onPress, disabled }: ChipProps) {
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected, disabled }}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={[styles.chip, selected ? styles.selected : null]}
    >
      <Text tone={selected ? "inverse" : "default"} variant="label" weight="semibold">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated
  },
  selected: { backgroundColor: colors.primary, borderColor: colors.primary }
});
