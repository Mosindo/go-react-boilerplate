import React, { type ComponentProps } from "react";
import { Pressable, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";

type IconButtonProps = {
  icon: ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  size?: number;
  tone?: "default" | "primary" | "muted" | "onMedia";
  filled?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({ icon, label, onPress, size = 44, tone = "default", filled, disabled, style, testID }: IconButtonProps) {
  const theme = useTheme();
  const c = theme.colors;
  const fg = tone === "primary" ? (filled ? c.onPrimary : c.primary) : tone === "muted" ? c.textMuted : tone === "onMedia" ? "#FFFFFF" : c.text;
  const bg = filled ? (tone === "primary" ? c.primary : c.surface) : tone === "onMedia" ? "rgba(0,0,0,0.35)" : "transparent";
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
          backgroundColor: bg,
          opacity: disabled ? 0.4 : pressed ? 0.7 : 1
        },
        filled ? theme.shadow(2) : null,
        style
      ]}
    >
      <Ionicons color={fg} name={icon} size={Math.round(size * 0.48)} />
    </Pressable>
  );
}
