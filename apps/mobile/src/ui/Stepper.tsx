import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  onChange: (value: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, suffix, onChange }: Props) {
  return (
    <View style={styles.row} accessibilityRole="adjustable" accessibilityLabel={label} accessibilityValue={{ min, max, now: value }}>
      <Text variant="label" style={styles.label}>
        {label}
      </Text>
      <IconButton
        icon="remove"
        label={`Diminuer ${label}`}
        size={36}
        disabled={value <= min}
        onPress={() => onChange(Math.max(min, value - step))}
      />
      <Text variant="heading" style={styles.value}>
        {value}
        {suffix ?? ""}
      </Text>
      <IconButton
        icon="add"
        label={`Augmenter ${label}`}
        size={36}
        disabled={value >= max}
        onPress={() => onChange(Math.min(max, value + step))}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.xs },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
