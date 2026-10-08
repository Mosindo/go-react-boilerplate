import React from "react";
import { StyleSheet, View } from "react-native";
import { clamp } from "../../utils/validation";
import { IconButton } from "./Button";
import { Text } from "./Text";
import { spacing } from "./tokens";
import { useTheme } from "../../theme";

type StepperProps = { label: string; value: number; min: number; max: number; onChange: (value: number) => void; step?: number };

/** Accessible numeric stepper, used instead of a slider (no extra native dependency, works on web). */
export function Stepper({ label, value, min, max, onChange, step = 1 }: StepperProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={`${label} : ${value}`} accessibilityRole="adjustable" style={styles.row}>
      <Text style={styles.label} tone="muted" variant="label">
        {label}
      </Text>
      <IconButton
        background={colors.surfaceMuted}
        disabled={value <= min}
        icon="remove"
        label={`Diminuer ${label}`}
        onPress={() => onChange(clamp(value - step, min, max))}
        size={40}
      />
      <Text align="center" style={styles.value} variant="heading">
        {value}
      </Text>
      <IconButton
        background={colors.surfaceMuted}
        disabled={value >= max}
        icon="add"
        label={`Augmenter ${label}`}
        onPress={() => onChange(clamp(value + step, min, max))}
        size={40}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  label: { flex: 1 },
  value: { minWidth: 44 }
});
