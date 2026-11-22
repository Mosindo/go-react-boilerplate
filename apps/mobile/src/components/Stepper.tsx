import React from "react";
import { StyleSheet, View } from "react-native";
import { IconButton, Text, colors, radii, spacing } from "../ui";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  onChange: (next: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, suffix, onChange }: Props) {
  return (
    <View style={styles.row}>
      <Text style={styles.label} variant="label">
        {label}
      </Text>
      <IconButton
        disabled={value <= min}
        icon="remove"
        label={`Diminuer ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
        size={36}
      />
      <Text accessibilityLiveRegion="polite" style={styles.value} variant="heading">
        {value}
        {suffix ?? ""}
      </Text>
      <IconButton
        disabled={value >= max}
        icon="add"
        label={`Augmenter ${label}`}
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
    padding: spacing.sm,
    paddingLeft: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  },
  label: {
    flex: 1
  },
  value: {
    minWidth: 56,
    textAlign: "center"
  }
});
