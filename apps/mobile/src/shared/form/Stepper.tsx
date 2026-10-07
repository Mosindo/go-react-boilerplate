import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { Text, radii, spacing } from "../ui";

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  unit?: string;
  onChange: (value: number) => void;
};

export function clampStep(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function Stepper({ label, max, min, onChange, step = 1, unit = "", value }: StepperProps) {
  const { colors } = useTheme();
  const buttonStyle = [
    styles.button,
    { borderColor: colors.borderStrong, backgroundColor: colors.backgroundElevated }
  ];

  return (
    <View style={styles.row}>
      <Text tone="secondary" variant="label" weight="semibold">
        {label}
      </Text>
      <View style={styles.controls}>
        <Pressable
          accessibilityLabel={`${label} -`}
          accessibilityRole="button"
          disabled={value <= min}
          onPress={() => onChange(clampStep(value - step, min, max))}
          style={[buttonStyle, value <= min ? styles.disabled : null]}
        >
          <Text variant="heading">-</Text>
        </Pressable>
        <Text style={styles.value} variant="body" weight="bold">
          {value}
          {unit}
        </Text>
        <Pressable
          accessibilityLabel={`${label} +`}
          accessibilityRole="button"
          disabled={value >= max}
          onPress={() => onChange(clampStep(value + step, min, max))}
          style={[buttonStyle, value >= max ? styles.disabled : null]}
        >
          <Text variant="heading">+</Text>
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
    alignItems: "center",
    justifyContent: "center"
  },
  disabled: { opacity: 0.4 },
  value: { minWidth: 64, textAlign: "center" }
});
