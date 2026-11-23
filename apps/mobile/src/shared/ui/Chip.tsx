import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
};

export function Chip({ disabled, label, onPress, selected = false, testID }: ChipProps) {
  const content = (
    <Text tone={selected ? "primary" : "muted"} variant="label" weight={selected ? "semibold" : "medium"}>
      {label}
    </Text>
  );
  if (!onPress) {
    return <Pressable style={[styles.base, selected ? styles.selected : null]}>{content}</Pressable>;
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        selected ? styles.selected : null,
        pressed ? styles.pressed : null,
        disabled ? styles.disabled : null
      ]}
      testID={testID}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  selected: {
    borderColor: colors.primaryBorder,
    backgroundColor: colors.primarySoft
  },
  pressed: {
    opacity: 0.8
  },
  disabled: {
    opacity: 0.4
  }
});
