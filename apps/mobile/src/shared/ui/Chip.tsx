import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { radii, spacing } from "./tokens";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
};

/** Selectable pill (interests, genders). Without onPress it is a static tag. */
export function Chip({ label, selected = false, onPress, disabled }: ChipProps) {
  const { colors } = useTheme();
  const box = {
    backgroundColor: selected ? colors.primarySoft : colors.surfaceMuted,
    borderColor: selected ? colors.primary : colors.border
  };
  const text = (
    <Text variant="label" weight="semibold" tone={selected ? "primary" : "default"}>
      {label}
    </Text>
  );
  if (!onPress) {
    return <View style={[styles.chip, box]}>{text}</View>;
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [styles.chip, box, pressed && { opacity: 0.8 }]}
    >
      {text}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    minHeight: 34,
    justifyContent: "center"
  }
});
