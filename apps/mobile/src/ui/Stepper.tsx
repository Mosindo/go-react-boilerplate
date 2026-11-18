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
    <View style={styles.row}>
      <Text style={styles.label} variant="bodyStrong">
        {label}
      </Text>
      <IconButton
        disabled={value <= min}
        icon="remove"
        label={`Diminuer ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
        size={38}
        tone="surface"
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
        size={38}
        tone="surface"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
