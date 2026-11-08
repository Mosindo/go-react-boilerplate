import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; disabled?: boolean; testID?: string };

export function Chip({ label, selected = false, onPress, disabled, testID }: Props) {
  const { colors } = useTheme();
  const content = (
    <Text variant="label" style={{ color: selected ? colors.onPrimary : colors.text }}>
      {label}
    </Text>
  );
  const boxStyle = [
    styles.chip,
    {
      backgroundColor: selected ? colors.primary : colors.surface,
      borderColor: selected ? colors.primary : colors.border,
      opacity: disabled ? 0.5 : 1
    }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={boxStyle}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected, disabled: !!disabled }}
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      style={boxStyle}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs + 2,
    marginRight: spacing.xs,
    marginBottom: spacing.xs
  }
});
