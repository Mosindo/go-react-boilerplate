import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

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

/** Accessible numeric control (+/-) that works the same on native and web. */
export function Stepper({ label, value, min, max, step = 1, unit, onChange, testID }: Props) {
  const { colors, radii } = useTheme();
  const clamp = (v: number) => Math.max(min, Math.min(max, v));
  return (
    <View
      testID={testID}
      style={[styles.wrap, { borderColor: colors.border, backgroundColor: colors.surface, borderRadius: radii.md }]}
      accessibilityRole="adjustable"
      accessibilityLabel={label}
      accessibilityValue={{ min, max, now: value, text: `${value}${unit ? ` ${unit}` : ""}` }}
      accessibilityActions={[{ name: "increment" }, { name: "decrement" }]}
      onAccessibilityAction={(e) => onChange(clamp(value + (e.nativeEvent.actionName === "increment" ? step : -step)))}
    >
      <Text tone="muted" style={styles.label}>
        {label}
      </Text>
      <IconButton
        icon="remove"
        label={`Diminuer ${label}`}
        size={36}
        onPress={() => onChange(clamp(value - step))}
        disabled={value <= min}
      />
      <Text variant="heading" style={styles.value}>
        {value}
        {unit ? <Text tone="muted">{` ${unit}`}</Text> : null}
      </Text>
      <IconButton
        icon="add"
        label={`Augmenter ${label}`}
        size={36}
        onPress={() => onChange(clamp(value + step))}
        disabled={value >= max}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    paddingLeft: 16,
    paddingRight: 6,
    minHeight: 56,
    gap: 4
  },
  label: { flex: 1 },
  value: { minWidth: 72, textAlign: "center" }
});
