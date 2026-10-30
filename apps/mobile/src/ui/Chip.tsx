import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { Text } from "./Text";
import { MIN_TOUCH, radii, spacing, useTheme } from "./theme";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected = false, onPress, testID }: ChipProps) {
  const { colors } = useTheme();
  const content = (
    <Text variant="label" style={{ color: selected ? colors.primaryText : colors.text }}>
      {label}
    </Text>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: selected ? colors.primary : colors.surface,
      borderColor: selected ? colors.primary : colors.border
    }
  ];

  if (!onPress) {
    return (
      <Pressable disabled style={[...style, styles.static]}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected }}
      onPress={onPress}
      style={style}
      testID={testID}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    minHeight: MIN_TOUCH - 6,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.pill,
    borderWidth: 1,
    justifyContent: "center"
  },
  static: { minHeight: 32, paddingHorizontal: spacing.md }
});
