import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text, MIN_TOUCH, radius, spacing, useColors } from "../../shared/ui";

type Props = { label: string; value: number; min: number; max: number; step?: number; unit?: string; onChange: (v: number) => void; testID?: string };

export function Stepper({ label, value, min, max, step = 1, unit = "", onChange, testID }: Props) {
  const c = useColors();
  const button = (sign: -1 | 1, glyph: string, description: string) => {
    const next = value + sign * step;
    const disabled = next < min || next > max;
    return (
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${description} ${label}`}
        disabled={disabled}
        onPress={() => onChange(next)}
        style={[styles.btn, { borderColor: c.border, backgroundColor: c.surface, opacity: disabled ? 0.4 : 1 }]}
        testID={testID ? `${testID}-${sign < 0 ? "minus" : "plus"}` : undefined}
      >
        <Text variant="heading">{glyph}</Text>
      </Pressable>
    );
  };
  return (
    <View style={styles.row}>
      <Text variant="label" tone="muted" style={{ flex: 1 }}>{label}</Text>
      {button(-1, "−", "Diminuer")}
      <Text variant="heading" style={styles.value} accessibilityLabel={`${label} : ${value}${unit}`}>{value}{unit}</Text>
      {button(1, "+", "Augmenter")}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  btn: { width: MIN_TOUCH, height: MIN_TOUCH, borderRadius: radius.pill, borderWidth: 1.5, alignItems: "center", justifyContent: "center" },
  value: { minWidth: 72, textAlign: "center" },
});
