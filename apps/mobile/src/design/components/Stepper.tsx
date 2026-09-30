import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../ThemeProvider";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

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

/** Accessible numeric control (used instead of sliders for age/distance). */
export function Stepper({ label, value, min, max, step = 1, unit, onChange, testID }: Props) {
  const { colors, radii } = useTheme();
  const clamp = (v: number) => Math.min(max, Math.max(min, v));
  return (
    <View
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      accessibilityLabel={label}
      accessibilityRole="adjustable"
      accessibilityValue={{ min, max, now: value, text: `${value}${unit ? " " + unit : ""}` }}
      onAccessibilityAction={(e) => onChange(clamp(value + (e.nativeEvent.actionName === "increment" ? step : -step)))}
      style={[styles.row, { backgroundColor: colors.surface, borderColor: colors.border, borderRadius: radii.md }]}
      testID={testID}
    >
      <Text style={styles.label} variant="label">
        {label}
      </Text>
      <IconButton disabled={value <= min} icon="remove" label={`Diminuer ${label}`} onPress={() => onChange(clamp(value - step))} size={36} />
      <Text style={styles.value} variant="heading">
        {value}
        {unit ? <Text tone="muted" variant="caption">{` ${unit}`}</Text> : null}
      </Text>
      <IconButton disabled={value >= max} icon="add" label={`Augmenter ${label}`} onPress={() => onChange(clamp(value + step))} size={36} />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", paddingHorizontal: 14, paddingVertical: 8, borderWidth: 1, gap: 6 },
  label: { flex: 1 },
  value: { minWidth: 64, textAlign: "center" }
});
