import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { IconButton } from "./Button";
import { Text } from "./Text";

/** Accessible numeric control (no slider dependency). */
export function Stepper({
  label,
  value,
  min,
  max,
  step = 1,
  format,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (v: number) => string;
  onChange: (v: number) => void;
}) {
  const { colors } = useTheme();
  const display = format ? format(value) : String(value);
  return (
    <View
      style={[styles.row, { borderColor: colors.border, backgroundColor: colors.surface }]}
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={label}
      accessibilityValue={{ text: display }}
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      onAccessibilityAction={(e) => {
        if (e.nativeEvent.actionName === "increment") onChange(Math.min(max, value + step));
        if (e.nativeEvent.actionName === "decrement") onChange(Math.max(min, value - step));
      }}
    >
      <Text variant="label" tone="muted" style={styles.label}>
        {label}
      </Text>
      <IconButton icon="remove" label={`Diminuer ${label}`} size={38} disabled={value <= min} onPress={() => onChange(Math.max(min, value - step))} />
      <Text variant="bodyStrong" style={styles.value}>
        {display}
      </Text>
      <IconButton icon="add" label={`Augmenter ${label}`} size={38} disabled={value >= max} onPress={() => onChange(Math.min(max, value + step))} />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", borderWidth: 1, borderRadius: 14, paddingLeft: spacing.md, paddingRight: spacing.xxs, minHeight: 54 },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
