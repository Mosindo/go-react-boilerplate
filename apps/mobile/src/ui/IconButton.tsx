import React from "react";
import { Pressable, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "primary" | "muted" | "onImage";
  filled?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({
  icon,
  label,
  onPress,
  size = 44,
  tone = "default",
  filled,
  disabled,
  style,
  testID
}: Props) {
  const { colors, scheme } = useTheme();
  const fg = {
    default: colors.text,
    primary: filled ? colors.onPrimary : colors.primary,
    muted: colors.textMuted,
    onImage: "#FFFFFF"
  }[tone];
  const bg = filled
    ? tone === "primary"
      ? colors.primary
      : colors.surfaceRaised
    : tone === "onImage"
      ? "rgba(0,0,0,0.35)"
      : "transparent";
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: bg,
          opacity: disabled ? 0.4 : pressed ? 0.75 : 1
        },
        filled
          ? {
              borderWidth: tone === "primary" ? 0 : 1,
              borderColor: colors.border,
              boxShadow: `0 6px 18px ${scheme === "dark" ? "rgba(0,0,0,0.5)" : "rgba(59,42,36,0.16)"}`
            }
          : null,
        style
      ]}
    >
      <Ionicons name={icon} size={Math.round(size * 0.5)} color={fg} />
    </Pressable>
  );
}
