import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
};

export function Chip({ disabled, label, onPress, selected = false, testID }: ChipProps) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected, disabled }}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={[styles.chip, selected ? styles.selected : null, disabled ? styles.disabled : null]}
      testID={testID}
    >
      <Text tone={selected ? "primary" : "default"} variant="label" weight={selected ? "bold" : "medium"}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    minHeight: 36,
    paddingHorizontal: spacing.md,
    justifyContent: "center",
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated
  },
  selected: { backgroundColor: colors.primarySoft, borderColor: colors.primary },
  disabled: { opacity: 0.5 }
});
