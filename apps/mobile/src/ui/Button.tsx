import React, { type ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  loading?: boolean;
  icon?: ReactNode;
  style?: StyleProp<ViewStyle>;
  compact?: boolean;
};

export function Button({ label, variant = "primary", loading, disabled, icon, style, compact, ...props }: ButtonProps) {
  const isDisabled = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      aria-disabled={!!isDisabled}
      aria-busy={!!loading}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        compact ? styles.compact : null,
        variantStyles[variant],
        pressed && !isDisabled ? pressedStyles[variant] : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={variant === "primary" ? colors.accentText : colors.text} size="small" /> : icon}
        <Text variant="bodyStrong" tone={variant === "primary" ? "inverse" : variant === "danger" ? "danger" : "default"}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center"
  },
  compact: {
    minHeight: 40,
    paddingHorizontal: spacing.lg
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  disabled: {
    opacity: 0.5
  }
});

const variantStyles: Record<Variant, ViewStyle> = {
  primary: { backgroundColor: colors.accent },
  secondary: { backgroundColor: colors.surfaceRaised, borderWidth: 1, borderColor: colors.border },
  ghost: { backgroundColor: "transparent" },
  danger: { backgroundColor: colors.dangerSoft, borderWidth: 1, borderColor: colors.danger }
};

const pressedStyles: Record<Variant, ViewStyle> = {
  primary: { backgroundColor: colors.accentPressed },
  secondary: { backgroundColor: colors.surfacePressed },
  ghost: { backgroundColor: colors.surface },
  danger: { opacity: 0.85 }
};
