import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
  testID?: string;
};

export function Stepper({ label, value, min, max, onChange, testID }: StepperProps) {
  const step = (delta: number) => onChange(Math.min(max, Math.max(min, value + delta)));
  return (
    <View style={styles.row} testID={testID}>
      <Text tone="secondary" variant="label" weight="semibold">
        {label}
      </Text>
      <View style={styles.controls}>
        <Pressable
          accessibilityLabel={`Diminuer ${label}`}
          accessibilityRole="button"
          disabled={value <= min}
          onPress={() => step(-1)}
          style={[styles.button, value <= min ? styles.disabled : null]}
          testID={testID ? `${testID}-minus` : undefined}
        >
          <Text weight="bold">−</Text>
        </Pressable>
        <Text style={styles.value} weight="bold">
          {value}
        </Text>
        <Pressable
          accessibilityLabel={`Augmenter ${label}`}
          accessibilityRole="button"
          disabled={value >= max}
          onPress={() => step(1)}
          style={[styles.button, value >= max ? styles.disabled : null]}
          testID={testID ? `${testID}-plus` : undefined}
        >
          <Text weight="bold">+</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md },
  controls: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  button: {
    width: 44,
    height: 44,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    alignItems: "center",
    justifyContent: "center"
  },
  disabled: { opacity: 0.4 },
  value: { minWidth: 36, textAlign: "center" }
});
