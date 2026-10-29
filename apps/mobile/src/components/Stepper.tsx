import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  onChange: (value: number) => void;
};

export function Stepper({ label, value, min, max, step = 1, suffix = "", onChange }: Props) {
  const theme = useTheme();
  const button = (icon: "remove" | "add", delta: number, disabled: boolean, name: string) => (
    <Pressable
      accessibilityLabel={name}
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      hitSlop={6}
      onPress={() => onChange(Math.min(max, Math.max(min, value + delta)))}
      style={[
        styles.button,
        { borderColor: theme.border, backgroundColor: theme.surface, opacity: disabled ? 0.4 : 1 }
      ]}
    >
      <Ionicons color={theme.text} name={icon} size={20} />
    </Pressable>
  );
  return (
    <View style={styles.row}>
      <Text style={[styles.label, { color: theme.text }]}>{label}</Text>
      <View style={styles.controls}>
        {button("remove", -step, value <= min, `${label} : diminuer`)}
        <Text
          accessibilityLabel={`${label} : ${value}${suffix}`}
          style={[styles.value, { color: theme.text }]}
        >
          {value}
          {suffix}
        </Text>
        {button("add", step, value >= max, `${label} : augmenter`)}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: spacing.lg
  },
  label: { fontSize: fontSizes.md, fontWeight: "600" },
  controls: { flexDirection: "row", alignItems: "center" },
  button: {
    width: 40,
    height: 40,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  value: { minWidth: 72, textAlign: "center", fontSize: fontSizes.md, fontWeight: "700" }
});
