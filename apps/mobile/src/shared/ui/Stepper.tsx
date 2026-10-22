import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { colors, radii, spacing } from "./tokens";

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

/** +/- control used where a slider would be hard to drive precisely or to make accessible. */
export function Stepper({ label, value, min, max, step = 1, format, onChange, testID }: StepperProps) {
  const set = (next: number) => onChange(Math.min(max, Math.max(min, next)));
  return (
    <View style={styles.row} testID={testID}>
      <Text tone="secondary" variant="label" weight="semibold">
        {label}
      </Text>
      <View style={styles.controls}>
        <Pressable
          accessibilityLabel={`Diminuer ${label}`}
          accessibilityRole="button"
          accessibilityState={{ disabled: value <= min }}
          disabled={value <= min}
          onPress={() => set(value - step)}
          style={[styles.button, value <= min ? styles.disabled : null]}
          testID={testID ? `${testID}-minus` : undefined}
        >
          <Text weight="bold">−</Text>
        </Pressable>
        <Text accessibilityLiveRegion="polite" style={styles.value} weight="semibold">
          {format ? format(value) : String(value)}
        </Text>
        <Pressable
          accessibilityLabel={`Augmenter ${label}`}
          accessibilityRole="button"
          accessibilityState={{ disabled: value >= max }}
          disabled={value >= max}
          onPress={() => set(value + step)}
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
  controls: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  button: {
    width: 44,
    height: 44,
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
