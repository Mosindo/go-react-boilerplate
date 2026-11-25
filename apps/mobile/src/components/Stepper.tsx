import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, radii, spacing } from "../theme";
import { IconButton, Text } from "../ui";

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (v: number) => string;
  onChange: (v: number) => void;
  testID?: string;
};

/** Accessible numeric control (used instead of sliders). */
export function Stepper({ label, value, min, max, step = 1, format = String, onChange, testID }: StepperProps) {
  const set = (v: number) => onChange(Math.min(max, Math.max(min, v)));
  return (
    <View style={styles.row} accessible accessibilityLabel={`${label} : ${format(value)}`} testID={testID}>
      <Text variant="bodyStrong" style={styles.label}>
        {label}
      </Text>
      <IconButton
        icon="remove"
        label={`Diminuer ${label}`}
        onPress={() => set(value - step)}
        disabled={value <= min}
        background={colors.surfaceHigh}
        size={36}
      />
      <Text variant="bodyStrong" style={styles.value}>
        {format(value)}
      </Text>
      <IconButton
        icon="add"
        label={`Augmenter ${label}`}
        onPress={() => set(value + step)}
        disabled={value >= max}
        background={colors.surfaceHigh}
        size={36}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    backgroundColor: colors.surface,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  label: {
    flex: 1
  },
  value: {
    minWidth: 64,
    textAlign: "center"
  }
});
