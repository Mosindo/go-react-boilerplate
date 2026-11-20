import React from "react";
import { Pressable, type StyleProp, type ViewStyle } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme } from "../theme";

type Props = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "primary" | "danger" | "success";
  filled?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "default", filled, disabled, style, testID }: Props) {
  const { colors } = useTheme();
  const color = { default: colors.text, primary: colors.primary, danger: colors.danger, success: colors.success }[tone];
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: filled ? colors.surfaceRaised : pressed ? colors.surfaceAlt : "transparent",
          borderWidth: filled ? 1 : 0,
          borderColor: colors.border,
          opacity: disabled ? 0.4 : 1,
          transform: [{ scale: pressed ? 0.94 : 1 }]
        },
        style
      ]}
    >
      <Ionicons color={color} name={icon} size={Math.round(size * 0.48)} />
    </Pressable>
  );
}
