import React from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type OptionCardProps = {
  label: string;
  selected: boolean;
  onPress: () => void;
  /** "radio" for single choice, "check" for multiple. */
  mode?: "radio" | "check";
  testID?: string;
};

export function OptionCard({ label, mode = "radio", onPress, selected, testID }: OptionCardProps) {
  const palette = useTheme();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole={mode === "radio" ? "radio" : "checkbox"}
      accessibilityState={mode === "radio" ? { selected } : { checked: selected }}
      onPress={onPress}
      style={({ pressed }) => ({
        minHeight: 56,
        flexDirection: "row",
        alignItems: "center",
        justifyContent: "space-between",
        gap: spacing.md,
        paddingHorizontal: spacing.lg,
        borderRadius: radii.md,
        borderWidth: 1.5,
        borderColor: selected ? palette.brand : palette.border,
        backgroundColor: selected ? palette.brandSoft : palette.surface,
        opacity: pressed ? 0.85 : 1
      })}
      testID={testID}
    >
      <Text weight={selected ? "bold" : "medium"}>{label}</Text>
      <View
        style={{
          width: 24,
          height: 24,
          borderRadius: mode === "radio" ? 12 : 7,
          borderWidth: 2,
          borderColor: selected ? palette.brand : palette.borderStrong,
          backgroundColor: selected ? palette.brand : "transparent",
          alignItems: "center",
          justifyContent: "center"
        }}
      >
        {selected ? <Ionicons color={palette.isDark ? "#1B1017" : "#FFFFFF"} name="checkmark" size={16} /> : null}
      </View>
    </Pressable>
  );
}
