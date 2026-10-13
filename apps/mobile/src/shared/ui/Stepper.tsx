import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { spacing } from "./tokens";

type StepperProps = {
  value: number;
  min: number;
  max: number;
  label: string;
  onChange: (value: number) => void;
};

export function Stepper({ value, min, max, label, onChange }: StepperProps) {
  const { colors } = useTheme();
  const button = (name: "remove" | "add", next: number, disabled: boolean, a11y: string) => (
    <Pressable
      accessibilityLabel={a11y}
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      hitSlop={8}
      onPress={() => onChange(next)}
      style={[styles.button, { borderColor: colors.border, opacity: disabled ? 0.4 : 1, backgroundColor: colors.surface }]}
    >
      <Ionicons color={colors.text} name={name} size={20} />
    </Pressable>
  );
  return (
    <View accessibilityLabel={`${label} : ${value}`} style={styles.root}>
      {button("remove", value - 1, value <= min, `Diminuer ${label}`)}
      <Text style={styles.value} variant="heading" weight="bold">
        {value}
      </Text>
      {button("add", value + 1, value >= max, `Augmenter ${label}`)}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  button: { width: 40, height: 40, borderRadius: 20, borderWidth: 1.5, alignItems: "center", justifyContent: "center" },
  value: { minWidth: 36, textAlign: "center" }
});
