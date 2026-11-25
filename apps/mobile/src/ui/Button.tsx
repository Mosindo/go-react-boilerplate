import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: Variant;
  icon?: keyof typeof Ionicons.glyphMap;
  loading?: boolean;
  disabled?: boolean;
  compact?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

const variantStyles: Record<Variant, { bg: string; pressed: string; fg: keyof typeof colors; border?: string }> = {
  primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: "onPrimary" },
  secondary: { bg: colors.surfaceHigh, pressed: colors.border, fg: "text", border: colors.borderStrong },
  ghost: { bg: "transparent", pressed: colors.surface, fg: "textMuted" },
  danger: { bg: colors.dangerSoft, pressed: "rgba(242,114,114,0.24)", fg: "danger", border: "rgba(242,114,114,0.4)" }
};

export function Button({ label, onPress, variant = "primary", icon, loading, disabled, compact, style, testID }: ButtonProps) {
  const v = variantStyles[variant];
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        compact && styles.compact,
        { backgroundColor: pressed ? v.pressed : v.bg, borderColor: v.border ?? "transparent" },
        inactive && styles.disabled,
        style
      ]}
    >
      <View style={styles.content}>
        {loading ? (
          <ActivityIndicator color={colors[v.fg]} />
        ) : (
          <>
            {icon ? <Ionicons name={icon} size={18} color={colors[v.fg]} /> : null}
            <Text variant="bodyStrong" color={v.fg}>
              {label}
            </Text>
          </>
        )}
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 52,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.lg,
    justifyContent: "center",
    borderWidth: 1
  },
  compact: {
    minHeight: 40,
    paddingHorizontal: spacing.md
  },
  content: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xs
  },
  disabled: {
    opacity: 0.5
  }
});
