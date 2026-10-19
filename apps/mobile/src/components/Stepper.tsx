import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { MIN_TOUCH, radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "../ui";

type Props = { label: string; value: number; min: number; max: number; onChange: (value: number) => void };

export function Stepper({ label, value, min, max, onChange }: Props) {
  const { colors } = useTheme();
  const button = (symbol: string, name: string, next: number, disabled: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${name} ${label}`}
      disabled={disabled}
      onPress={() => onChange(next)}
      style={[styles.button, { backgroundColor: colors.surfaceAlt, opacity: disabled ? 0.4 : 1 }]}
    >
      <Text variant="subtitle">{symbol}</Text>
    </Pressable>
  );
  return (
    <View style={styles.row}>
      <Text style={styles.label}>{label}</Text>
      {button("−", "Diminuer", value - 1, value <= min)}
      <Text bold style={styles.value} accessibilityLabel={`${label} : ${value}`}>
        {value}
      </Text>
      {button("+", "Augmenter", value + 1, value >= max)}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  label: { flex: 1 },
  button: {
    width: MIN_TOUCH,
    height: MIN_TOUCH,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center"
  },
  value: { minWidth: 32, textAlign: "center" }
});
