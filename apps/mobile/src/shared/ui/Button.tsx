import React, { type ReactNode } from "react";
import { ActivityIndicator, Pressable, View, type PressableProps, type StyleProp, type TextStyle, type ViewStyle } from "react-native";
import { elevation, useTheme } from "../../theme";
import { Text } from "./Text";
import { controls, radii, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  icon?: ReactNode;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
};

export function Button({
  children,
  disabled,
  fullWidth = false,
  icon,
  label,
  loading = false,
  size = "md",
  style,
  textStyle,
  variant = "primary",
  accessibilityLabel,
  ...props
}: ButtonProps) {
  const palette = useTheme();
  const inactive = disabled || loading;
  const text = typeof children === "string" ? children : label;

  const variants: Record<ButtonVariant, { bg: string; pressedBg: string; fg: string; border: string }> = {
    primary: { bg: palette.primary, pressedBg: palette.primaryPressed, fg: palette.onPrimary, border: "transparent" },
    secondary: { bg: palette.brandSoft, pressedBg: palette.surfaceMuted, fg: palette.brand, border: "transparent" },
    outline: { bg: "transparent", pressedBg: palette.surfaceMuted, fg: palette.text, border: palette.borderStrong },
    ghost: { bg: "transparent", pressedBg: palette.surfaceMuted, fg: palette.brand, border: "transparent" },
    destructive: { bg: palette.danger, pressedBg: palette.danger, fg: palette.isDark ? "#1B1017" : "#FFFFFF", border: "transparent" }
  };
  const v = variants[variant];

  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? text}
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        {
          minHeight: controls.button[size],
          borderRadius: radii.pill,
          paddingHorizontal: size === "sm" ? spacing.lg : spacing.xl,
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: pressed ? v.pressedBg : v.bg,
          borderWidth: variant === "outline" ? 1.5 : 0,
          borderColor: v.border,
          opacity: inactive ? 0.55 : 1,
          alignSelf: fullWidth ? "stretch" : "auto"
        },
        variant === "primary" && !inactive ? elevation(palette) : null,
        style
      ]}
      {...props}
    >
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.sm }}>
        {loading ? <ActivityIndicator color={v.fg} size="small" /> : icon}
        {text ? (
          <Text style={[{ color: v.fg }, textStyle]} variant="button" weight="semibold">
            {text}
          </Text>
        ) : null}
      </View>
    </Pressable>
  );
}
