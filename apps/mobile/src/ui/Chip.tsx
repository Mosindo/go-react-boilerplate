import React from "react";
import { Pressable, StyleSheet } from "react-native";

import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export interface ChipProps {
  label: string;
  selected?: boolean;
  /** Highlights an interest shared with the viewer. */
  highlight?: boolean;
  onPress?: () => void;
  testID?: string;
}

export function Chip({ label, selected = false, highlight = false, onPress, testID }: ChipProps) {
  const { colors } = useTheme();
  const active = selected || highlight;
  return (
    <Pressable
      testID={testID}
      disabled={!onPress}
      onPress={onPress}
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected }}
      style={[
        styles.chip,
        {
          backgroundColor: selected
            ? colors.primary
            : highlight
              ? colors.primarySoft
              : colors.surface,
          borderColor: active ? colors.primary : colors.border,
        },
      ]}
    >
      <Text
        variant="label"
        style={{ color: selected ? colors.onPrimary : highlight ? colors.primary : colors.text }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderRadius: radii.pill,
    borderWidth: 1.5,
  },
});
