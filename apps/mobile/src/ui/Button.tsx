import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme } from "../theme";
import { Text } from "./Text";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type Props = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: ButtonVariant;
  icon?: keyof typeof Ionicons.glyphMap;
  loading?: boolean;
  size?: "md" | "sm";
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, variant = "primary", icon, loading, disabled, size = "md", style, ...props }: Props) {
  const { colors, radii, spacing } = useTheme();
  const isDisabled = disabled || loading;
  const palette = {
    primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.surfaceAlt, pressed: colors.border, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", pressed: colors.surfaceAlt, fg: colors.primary, border: "transparent" },
    danger: { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: colors.danger, border: colors.danger }
  }[variant];

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: size === "md" ? 52 : 40,
          paddingHorizontal: size === "md" ? spacing.xl : spacing.md,
          borderRadius: radii.pill,
          backgroundColor: pressed ? palette.pressed : palette.bg,
          borderColor: palette.border,
          opacity: isDisabled ? 0.55 : 1,
          transform: [{ scale: pressed ? 0.98 : 1 }]
        },
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? (
          <ActivityIndicator color={palette.fg} size="small" />
        ) : icon ? (
          <Ionicons color={palette.fg} name={icon} size={size === "md" ? 20 : 16} />
        ) : null}
        <Text style={{ color: palette.fg }} variant="label">
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", borderWidth: 1 },
  content: { flexDirection: "row", alignItems: "center", gap: 8 }
});
