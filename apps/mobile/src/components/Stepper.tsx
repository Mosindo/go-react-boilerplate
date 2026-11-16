import React from "react";
import { StyleSheet, View } from "react-native";
import { colors, radii, spacing } from "../theme/tokens";
import { IconButton, Text } from "../ui";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  unit?: string;
  onChange: (v: number) => void;
  testID?: string;
};

export function Stepper({ label, value, min, max, step = 1, unit, onChange, testID }: Props) {
  const clamp = (v: number) => Math.max(min, Math.min(max, v));
  return (
    <View
      style={styles.row}
      testID={testID}
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={label}
      accessibilityValue={{ min, max, now: value, text: `${value}${unit ?? ""}` }}
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      onAccessibilityAction={(e) => onChange(clamp(value + (e.nativeEvent.actionName === "increment" ? step : -step)))}
    >
      <Text variant="bodyStrong" style={styles.label}>
        {label}
      </Text>
      <IconButton
        icon="remove"
        label={`Diminuer ${label}`}
        size={38}
        disabled={value <= min}
        onPress={() => onChange(clamp(value - step))}
        testID={testID ? `${testID}-minus` : undefined}
      />
      <Text variant="heading" style={styles.value}>
        {value}
        {unit ?? ""}
      </Text>
      <IconButton
        icon="add"
        label={`Augmenter ${label}`}
        size={38}
        disabled={value >= max}
        onPress={() => onChange(clamp(value + step))}
        testID={testID ? `${testID}-plus` : undefined}
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
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.md,
    paddingLeft: spacing.lg
  },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
