import React, { type ComponentProps, type ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../../theme";
import { Text } from "./Text";
import { controls, radii, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";
export type ButtonSize = "sm" | "md" | "lg";
export type IconName = ComponentProps<typeof Ionicons>["name"];

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  icon?: IconName;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
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
  variant = "primary",
  ...props
}: ButtonProps) {
  const { colors } = useTheme();
  const inactive = disabled || loading;
  const text = typeof children === "string" ? children : label;
  const palette = {
    primary: { bg: colors.primary, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primarySoft },
    outline: { bg: "transparent", fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    destructive: { bg: colors.danger, fg: "#FFFFFF", border: colors.danger }
  }[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={props.accessibilityLabel ?? text}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        { minHeight: controls.button[size], backgroundColor: palette.bg, borderColor: palette.border },
        size === "sm" ? styles.compact : null,
        fullWidth ? styles.fullWidth : null,
        inactive ? styles.disabled : null,
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? (
          <ActivityIndicator color={palette.fg} size="small" />
        ) : icon ? (
          <Ionicons color={palette.fg} name={icon} size={size === "sm" ? 16 : 20} />
        ) : null}
        {text ? (
          <Text style={{ color: palette.fg }} variant="button">
            {text}
          </Text>
        ) : null}
      </View>
    </Pressable>
  );
}

export type IconButtonProps = Omit<PressableProps, "style" | "children"> & {
  icon: IconName;
  label: string;
  size?: number;
  color?: string;
  background?: string;
  style?: StyleProp<ViewStyle>;
};

/** Round icon-only button; `label` is mandatory because it is the accessible name. */
export function IconButton({ icon, label, size = 44, color, background, style, ...props }: IconButtonProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      hitSlop={6}
      style={({ pressed }) => [
        styles.icon,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: background ?? "transparent" },
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <Ionicons color={color ?? colors.text} name={icon} size={Math.round(size * 0.5)} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { borderRadius: radii.pill, borderWidth: 1, alignItems: "center", justifyContent: "center", paddingHorizontal: spacing.xl },
  compact: { paddingHorizontal: spacing.lg },
  fullWidth: { alignSelf: "stretch" },
  disabled: { opacity: 0.5 },
  pressed: { opacity: 0.85, transform: [{ scale: 0.98 }] },
  content: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.sm },
  icon: { alignItems: "center", justifyContent: "center" }
});
