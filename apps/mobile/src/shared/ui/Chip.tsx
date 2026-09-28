import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { hitSlop, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export type ChipProps = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  disabled?: boolean;
};

/** Selectable pill (interests, genders). Without onPress it renders as a static tag. */
export function Chip({ label, selected = false, highlighted = false, onPress, disabled = false }: ChipProps) {
  const theme = useTheme();
  const active = selected || highlighted;
  const body = (
    <View
      style={[
        styles.chip,
        {
          backgroundColor: active ? theme.primarySoft : theme.surface,
          borderColor: active ? theme.primary : theme.border
        }
      ]}
    >
      <Text style={{ color: active ? theme.text : theme.textMuted }} variant="label">
        {label}
      </Text>
    </View>
  );
  if (!onPress) {
    return body;
  }
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={[styles.target, { opacity: disabled && !selected ? 0.5 : 1 }]}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  target: { minHeight: hitSlop, justifyContent: "center" },
  chip: {
    borderWidth: 1,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  }
});
