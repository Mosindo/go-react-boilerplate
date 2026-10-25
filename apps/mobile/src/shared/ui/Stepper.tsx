import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  format?: (value: number) => string;
  onChange: (value: number) => void;
  testID?: string;
};

export function Stepper({ label, value, min, max, step = 1, format, onChange, testID }: Props) {
  const shown = format ? format(value) : String(value);
  return (
    <View
      accessible
      accessibilityLabel={`${label} : ${shown}`}
      accessibilityRole="adjustable"
      style={styles.row}
      testID={testID}
    >
      <Text style={styles.label} variant="label">
        {label}
      </Text>
      <IconButton
        disabled={value <= min}
        icon="remove-circle-outline"
        label={`Diminuer ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
      />
      <Text style={styles.value} variant="label">
        {shown}
      </Text>
      <IconButton
        disabled={value >= max}
        icon="add-circle-outline"
        label={`Augmenter ${label}`}
        onPress={() => onChange(Math.min(max, value + step))}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
