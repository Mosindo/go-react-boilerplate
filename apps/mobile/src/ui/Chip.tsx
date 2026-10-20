import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export function Chip({
  label,
  selected = false,
  onPress,
  testID
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected }}
      accessibilityLabel={label}
      disabled={!onPress}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primary : colors.surfaceAlt,
          borderColor: selected ? colors.primary : colors.border
        }
      ]}
    >
      <Text variant="caption" style={{ color: selected ? colors.onPrimary : colors.text, fontWeight: "600" }}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    marginRight: spacing.sm,
    marginBottom: spacing.sm
  }
});
