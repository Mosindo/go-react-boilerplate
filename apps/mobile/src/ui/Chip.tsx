import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; disabled?: boolean };

/** A selectable pill (interests, genders) or a static tag when no onPress is given. */
export function Chip({ label, selected = false, onPress, disabled }: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected, disabled }}
      disabled={!onPress || disabled}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primarySoft : colors.surface,
          borderColor: selected ? colors.primary : colors.border,
          opacity: disabled ? 0.5 : 1
        }
      ]}
    >
      <Text variant="label" style={{ color: selected ? colors.primary : colors.text }}>
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
    paddingVertical: spacing.sm,
    minHeight: 36,
    justifyContent: "center"
  }
});
