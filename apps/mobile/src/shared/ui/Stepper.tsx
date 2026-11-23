import React from "react";
import { StyleSheet, View } from "react-native";
import { IconButton } from "./IconButton";
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

export function Stepper({ format, label, max, min, onChange, step = 1, testID, value }: StepperProps) {
  return (
    <View accessibilityLabel={label} style={styles.row} testID={testID}>
      <Text style={styles.label} tone="muted" variant="label">
        {label}
      </Text>
      <IconButton
        accessibilityLabel={`Diminuer ${label}`}
        disabled={value - step < min}
        glyph="−"
        onPress={() => onChange(Math.max(min, value - step))}
        size={36}
      />
      <Text accessibilityLiveRegion="polite" style={styles.value} weight="semibold">
        {format ? format(value) : String(value)}
      </Text>
      <IconButton
        accessibilityLabel={`Augmenter ${label}`}
        disabled={value + step > max}
        glyph="+"
        onPress={() => onChange(Math.min(max, value + step))}
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
    paddingVertical: spacing.xs,
    paddingHorizontal: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surfaceMuted,
    borderWidth: 1,
    borderColor: colors.border
  },
  label: {
    flex: 1
  },
  value: {
    minWidth: 64,
    textAlign: "center"
  }
});
