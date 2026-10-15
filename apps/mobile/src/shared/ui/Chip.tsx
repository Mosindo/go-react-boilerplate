import React from "react";
import { Pressable, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { radii, spacing } from "./tokens";

type ChipProps = {
  label: string;
  selected?: boolean;
  /** Static chips (read-only tags) are not pressable. */
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function Chip({ disabled = false, label, onPress, selected = false, style, testID }: ChipProps) {
  const palette = useTheme();
  const background = selected ? palette.brand : palette.surface;
  const foreground = selected ? (palette.isDark ? "#1B1017" : "#FFFFFF") : palette.text;

  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole={onPress ? "checkbox" : "text"}
      accessibilityState={onPress ? { checked: selected, disabled } : undefined}
      disabled={disabled || !onPress}
      onPress={onPress}
      style={({ pressed }) => [
        {
          flexDirection: "row",
          alignItems: "center",
          gap: spacing.xs,
          minHeight: onPress ? 40 : 32,
          paddingHorizontal: spacing.md,
          borderRadius: radii.pill,
          borderWidth: 1.5,
          borderColor: selected ? palette.brand : palette.border,
          backgroundColor: background,
          opacity: disabled ? 0.45 : pressed ? 0.8 : 1
        },
        style
      ]}
      testID={testID}
    >
      {selected && onPress ? <Ionicons color={foreground} name="checkmark" size={15} /> : null}
      <Text style={{ color: foreground }} variant="label" weight="medium">
        {label}
      </Text>
    </Pressable>
  );
}
