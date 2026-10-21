import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { MIN_TOUCH, radius, spacing, useTheme } from "../../theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string; disabled?: boolean };

/** Selectable pill (genders, interests) — or a static tag when onPress is omitted. */
export function Chip({ label, selected, onPress, testID, disabled }: Props) {
  const theme = useTheme();
  const bg = selected ? theme.primarySoft : theme.surface;
  const border = selected ? theme.primary : theme.border;
  return (
    <Pressable
      testID={testID}
      disabled={!onPress || disabled}
      onPress={onPress}
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected: !!selected, disabled: !!disabled }}
      accessibilityLabel={label}
      style={[styles.chip, { backgroundColor: bg, borderColor: border, opacity: disabled ? 0.5 : 1 }]}
    >
      <Text variant="caption" tone={selected ? "primary" : "default"} style={{ fontWeight: "600" }}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    minHeight: MIN_TOUCH - 4,
    justifyContent: "center",
    paddingHorizontal: spacing.md,
    borderRadius: radius.pill,
    borderWidth: 1
  }
});
