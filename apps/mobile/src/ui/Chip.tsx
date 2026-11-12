import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export function Chip({ label, selected = false, onPress, disabled }: { label: string; selected?: boolean; onPress?: () => void; disabled?: boolean }) {
  const { colors } = useTheme();
  const content = (
    <Text variant="label" style={{ color: selected ? colors.onPrimary : colors.text }}>
      {label}
    </Text>
  );
  const base = [styles.chip, { backgroundColor: selected ? colors.primary : colors.surface, borderColor: selected ? colors.primary : colors.border }];
  if (!onPress) {
    return (
      <Pressable style={[base, { backgroundColor: colors.surfaceMuted, borderColor: colors.surfaceMuted }]} disabled>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [...base, { opacity: disabled && !selected ? 0.45 : pressed ? 0.8 : 1 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { paddingHorizontal: spacing.md, paddingVertical: spacing.xs + 2, borderRadius: radii.pill, borderWidth: 1 }
});
