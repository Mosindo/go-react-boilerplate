import React from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  onChange: (value: number) => void;
  testID?: string;
};

export function Stepper({ label, max, min, onChange, step = 1, suffix = "", testID, value }: StepperProps) {
  const palette = useTheme();
  const button = (delta: number, icon: "remove" | "add", name: string) => {
    const next = value + delta;
    const disabled = next < min || next > max;
    return (
      <Pressable
        accessibilityLabel={`${name} ${label}`}
        accessibilityRole="button"
        accessibilityState={{ disabled }}
        disabled={disabled}
        onPress={() => onChange(next)}
        style={({ pressed }) => ({
          width: 44,
          height: 44,
          borderRadius: radii.pill,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: pressed ? palette.surfaceMuted : palette.surface,
          borderWidth: 1.5,
          borderColor: palette.borderStrong,
          opacity: disabled ? 0.35 : 1
        })}
      >
        <Ionicons color={palette.text} name={icon} size={22} />
      </Pressable>
    );
  };

  return (
    <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", minHeight: 56 }} testID={testID}>
      <Text weight="semibold">{label}</Text>
      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
        {button(-step, "remove", "Diminuer")}
        <Text accessibilityLiveRegion="polite" style={{ minWidth: 64, textAlign: "center" }} weight="bold">
          {value}
          {suffix}
        </Text>
        {button(step, "add", "Augmenter")}
      </View>
    </View>
  );
}
