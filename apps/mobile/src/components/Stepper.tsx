import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (value: number) => string;
  onChange: (value: number) => void;
  testID?: string;
};

export function Stepper({ label, value, min, max, step = 1, format, onChange, testID }: StepperProps) {
  const set = (next: number) => onChange(Math.min(Math.max(next, min), max));
  return (
    <View style={styles.row} testID={testID}>
      <Text style={styles.label} variant="label" weight="semibold">
        {label}
      </Text>
      <View style={styles.controls}>
        <Pressable
          accessibilityLabel={`Diminuer ${label}`}
          accessibilityRole="button"
          disabled={value <= min}
          onPress={() => set(value - step)}
          style={[styles.button, value <= min ? styles.disabled : null]}
        >
          <Text weight="bold">−</Text>
        </Pressable>
        <Text style={styles.value} weight="bold">
          {format ? format(value) : String(value)}
        </Text>
        <Pressable
          accessibilityLabel={`Augmenter ${label}`}
          accessibilityRole="button"
          disabled={value >= max}
          onPress={() => set(value + step)}
          style={[styles.button, value >= max ? styles.disabled : null]}
        >
          <Text weight="bold">+</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md },
  label: { flex: 1 },
  controls: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  button: {
    width: 40,
    height: 40,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated
  },
  disabled: { opacity: 0.4 },
  value: { minWidth: 72, textAlign: "center" }
});
