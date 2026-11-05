import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme";
import { Text } from "./Text";

type Props = {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
};

export function Stepper({ label, value, min, max, onChange }: Props) {
  const t = useTheme();
  const btn = (icon: "remove" | "add", delta: number, disabled: boolean) => (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${delta < 0 ? "Diminuer" : "Augmenter"} ${label}`}
      disabled={disabled}
      onPress={() => onChange(Math.min(max, Math.max(min, value + delta)))}
      style={[
        styles.btn,
        { borderColor: t.border, backgroundColor: t.surface, opacity: disabled ? 0.4 : 1 }
      ]}
    >
      <Ionicons name={icon} size={20} color={t.text} />
    </Pressable>
  );
  return (
    <View style={styles.row}>
      <Text variant="label">{label}</Text>
      <View style={styles.controls}>
        {btn("remove", -1, value <= min)}
        <Text variant="heading" style={styles.value} accessibilityLabel={`${label} : ${value}`}>
          {value}
        </Text>
        {btn("add", 1, value >= max)}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  controls: { flexDirection: "row", alignItems: "center", gap: 12 },
  btn: {
    width: 40,
    height: 40,
    borderRadius: 20,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  value: { minWidth: 32, textAlign: "center" }
});
