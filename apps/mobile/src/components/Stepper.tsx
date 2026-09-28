import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { IconButton, Text } from "../shared/ui";

export type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (value: number) => string;
  onChange: (value: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, format = String, onChange }: StepperProps) {
  return (
    <View style={styles.row}>
      <Text style={styles.label}>{label}</Text>
      <IconButton
        disabled={value <= min}
        glyph="−"
        label={`Decrease ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
      />
      <Text accessibilityLabel={`${label}: ${format(value)}`} center style={styles.value} variant="heading">
        {format(value)}
      </Text>
      <IconButton
        disabled={value >= max}
        glyph="+"
        label={`Increase ${label}`}
        onPress={() => onChange(Math.min(max, value + step))}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  label: { flex: 1 },
  value: { minWidth: 72 }
});
