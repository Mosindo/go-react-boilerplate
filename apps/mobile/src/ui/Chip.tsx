import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radii, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; disabled?: boolean };

export function Chip({ label, selected, onPress, disabled }: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected: !!selected, disabled: !!disabled }}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={[
        styles.chip,
        { backgroundColor: selected ? colors.primary : colors.surfaceAlt, borderColor: selected ? colors.primary : colors.border }
      ]}
    >
      <Text variant="caption" color={selected ? colors.onPrimary : colors.text} style={styles.text}>
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { paddingHorizontal: 14, paddingVertical: 8, borderRadius: radii.pill, borderWidth: 1 },
  text: { fontWeight: "600" }
});
