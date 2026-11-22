import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = Omit<PressableProps, "style" | "children"> & {
  label: string;
  variant?: Variant;
  loading?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  compact?: boolean;
  style?: StyleProp<ViewStyle>;
};

const backgrounds: Record<Variant, string> = {
  primary: colors.accent,
  secondary: colors.surfaceRaised,
  ghost: "transparent",
  danger: colors.dangerSoft
};

const foregrounds: Record<Variant, string> = {
  primary: colors.textOnAccent,
  secondary: colors.text,
  ghost: colors.textMuted,
  danger: colors.danger
};

export function Button({ label, variant = "primary", loading, icon, compact, disabled, style, ...props }: ButtonProps) {
  const isDisabled = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!isDisabled, busy: !!loading }}
      disabled={isDisabled}
      style={({ pressed }) => [
        styles.base,
        compact ? styles.compact : null,
        { backgroundColor: backgrounds[variant] },
        variant === "secondary" ? styles.bordered : null,
        pressed && !isDisabled ? styles.pressed : null,
        isDisabled ? styles.disabled : null,
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? (
          <ActivityIndicator color={foregrounds[variant]} size="small" />
        ) : icon ? (
          <Ionicons color={foregrounds[variant]} name={icon} size={18} />
        ) : null}
        <Text style={{ color: foregrounds[variant] }} variant="label">
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
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  compact: {
    minHeight: 40,
    paddingHorizontal: spacing.md
  },
  bordered: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.borderStrong
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  pressed: {
    opacity: 0.85,
    transform: [{ scale: 0.98 }]
  },
  disabled: {
    opacity: 0.5
  }
});
