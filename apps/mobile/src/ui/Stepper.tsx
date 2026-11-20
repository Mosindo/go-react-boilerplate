import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (value: number) => string;
  onChange: (value: number) => void;
};

/** An accessible numeric control (no slider dependency needed). */
export function Stepper({ label, value, min, max, step = 1, format, onChange }: Props) {
  const { colors, radii, spacing } = useTheme();
  const display = format ? format(value) : String(value);
  return (
    <View
      accessibilityLabel={`${label} : ${display}`}
      accessible={false}
      style={[styles.row, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.md, paddingHorizontal: spacing.sm }]}
    >
      <Text style={styles.label} tone="muted" variant="label">
        {label}
      </Text>
      <IconButton disabled={value <= min} icon="remove" label={`Diminuer ${label}`} onPress={() => onChange(Math.max(min, value - step))} size={36} />
      <Text style={styles.value} variant="bodyStrong">
        {display}
      </Text>
      <IconButton disabled={value >= max} icon="add" label={`Augmenter ${label}`} onPress={() => onChange(Math.min(max, value + step))} size={36} />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", minHeight: 56, borderWidth: 1, gap: 4 },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
