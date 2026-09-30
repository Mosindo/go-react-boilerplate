import React from "react";
import { Pressable, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../ThemeProvider";

type Props = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "primary" | "muted" | "onDark";
  filled?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "default", filled, disabled, style, testID }: Props) {
  const { colors } = useTheme();
  const color = {
    default: colors.text,
    primary: filled ? colors.onPrimary : colors.primary,
    muted: colors.textMuted,
    onDark: "#FFFFFF"
  }[tone];
  const background = filled ? (tone === "primary" ? colors.primary : colors.surfaceRaised) : "transparent";
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      style={({ pressed }) => [
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: background,
          opacity: disabled ? 0.4 : pressed ? 0.7 : 1,
          transform: [{ scale: pressed ? 0.94 : 1 }]
        },
        filled && {
          shadowColor: "#000",
          shadowOpacity: 0.12,
          shadowRadius: 12,
          shadowOffset: { width: 0, height: 6 },
          elevation: 4
        },
        style
      ]}
      testID={testID}
    >
      <Ionicons color={color} name={icon} size={Math.round(size * 0.48)} />
    </Pressable>
  );
}
