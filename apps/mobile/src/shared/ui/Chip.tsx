import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Pill used for interests, genders and filters. Selected state is exposed to assistive tech. */
export function Chip({ label, selected = false, onPress, disabled = false, style, testID }: ChipProps) {
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected, disabled }}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={[styles.base, selected ? styles.selected : null, disabled ? styles.disabled : null, style]}
      testID={testID}
    >
      <Text tone={selected ? "primary" : "default"} variant="label" weight={selected ? "bold" : "medium"}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2
  },
  selected: { backgroundColor: colors.primarySoft, borderColor: colors.primary },
  disabled: { opacity: 0.5 }
});
