import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, spacing } from "../theme/tokens";
import { AppText } from "./AppText";
import { IconButton } from "./IconButton";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  unit?: string;
  onChange: (value: number) => void;
  testID?: string;
};

/** Accessible numeric control, used instead of a slider dependency. */
export function Stepper({ label, value, min, max, step = 1, unit, onChange, testID }: Props) {
  const clamp = (v: number) => Math.min(max, Math.max(min, v));
  return (
    <View style={styles.row} accessibilityLabel={`${label} : ${value}${unit ? ` ${unit}` : ""}`} testID={testID}>
      <AppText style={styles.label}>{label}</AppText>
      <IconButton icon="remove" label={`Diminuer ${label}`} size={36} onPress={() => onChange(clamp(value - step))} disabled={value <= min} testID={testID ? `${testID}-minus` : undefined} />
      <AppText variant="bodyStrong" style={styles.value} align="center">
        {value}
        {unit ? ` ${unit}` : ""}
      </AppText>
      <IconButton icon="add" label={`Augmenter ${label}`} size={36} onPress={() => onChange(clamp(value + step))} disabled={value >= max} testID={testID ? `${testID}-plus` : undefined} />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.xs },
  label: { flex: 1, color: colors.text },
  value: { minWidth: 72 }
});
