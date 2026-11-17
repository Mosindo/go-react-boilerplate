import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, disabled, testID }: ChipProps) {
  const interactive = Boolean(onPress);
  return (
    <Pressable
      testID={testID}
      accessibilityRole={interactive ? "checkbox" : "text"}
      aria-checked={interactive ? !!selected : undefined}
      aria-disabled={interactive ? !!disabled : undefined}
      accessibilityLabel={label}
      disabled={!interactive || disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.chip,
        highlighted ? styles.highlighted : null,
        selected ? styles.selected : null,
        pressed ? styles.pressed : null,
        disabled && !selected ? styles.disabled : null
      ]}
    >
      <Text variant="label" tone={selected ? "inverse" : highlighted ? "gold" : "default"}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm - 1,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  selected: {
    backgroundColor: colors.accent,
    borderColor: colors.accent
  },
  highlighted: {
    borderColor: colors.gold,
    backgroundColor: colors.goldSoft
  },
  pressed: {
    opacity: 0.8
  },
  disabled: {
    opacity: 0.4
  }
});
