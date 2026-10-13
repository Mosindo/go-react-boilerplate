import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
};

/** Selectable pill (interests, genders). Without onPress it is a read-only tag. */
export function Chip({ label, selected = false, onPress, disabled, testID }: ChipProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole={onPress ? "checkbox" : "text"}
      accessibilityState={{ checked: selected, disabled }}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primarySoft : colors.surface,
          borderColor: selected ? colors.primary : colors.border,
          opacity: disabled ? 0.5 : 1
        }
      ]}
      testID={testID}
    >
      <Text tone={selected ? "primary" : "muted"} variant="label" weight={selected ? "bold" : "medium"}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1.5,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2
  }
});
