import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Icon } from "./Icon";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
};

export function Chip({ label, selected, onPress, disabled, testID }: ChipProps) {
  const { colors } = useTheme();
  const interactive = !!onPress;
  return (
    <Pressable
      accessibilityRole={interactive ? "checkbox" : "text"}
      accessibilityState={interactive ? { checked: !!selected, disabled: !!disabled } : undefined}
      disabled={!interactive || disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.chip,
        {
          backgroundColor: selected ? colors.primarySoft : colors.surfaceMuted,
          borderColor: selected ? colors.primary : "transparent",
          opacity: disabled && !selected ? 0.45 : pressed ? 0.8 : 1
        }
      ]}
      testID={testID}
    >
      {selected ? <Icon color={colors.primary} name="checkmark" size={16} /> : null}
      <Text style={{ color: selected ? colors.primary : colors.text }} variant="label">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    borderWidth: 1.5,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.sm,
    paddingVertical: 7
  }
});
