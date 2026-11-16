import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme/tokens";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = {
  label: string;
  onPress?: () => void;
  variant?: Variant;
  icon?: keyof typeof Ionicons.glyphMap;
  loading?: boolean;
  disabled?: boolean;
  compact?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  accessibilityHint?: string;
};

const palette: Record<Variant, { bg: string; pressed: string; fg: string; border: string }> = {
  primary: { bg: colors.accent, pressed: colors.accentPressed, fg: colors.textOnAccent, border: colors.accent },
  secondary: { bg: colors.surfaceRaised, pressed: colors.border, fg: colors.text, border: colors.border },
  ghost: { bg: colors.transparent, pressed: colors.surfaceRaised, fg: colors.text, border: colors.transparent },
  danger: { bg: colors.dangerSoft, pressed: "rgba(240,106,106,0.24)", fg: colors.danger, border: "rgba(240,106,106,0.35)" }
};

export function Button({ label, onPress, variant = "primary", icon, loading, disabled, compact, style, testID, accessibilityHint }: ButtonProps) {
  const p = palette[variant];
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityHint={accessibilityHint}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        compact && styles.compact,
        { backgroundColor: pressed ? p.pressed : p.bg, borderColor: p.border, opacity: inactive && !loading ? 0.5 : 1 },
        style
      ]}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={p.fg} size="small" /> : icon ? <Ionicons name={icon} size={compact ? 16 : 18} color={p.fg} /> : null}
        <Text variant="bodyStrong" style={{ color: p.fg }}>
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
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center"
  },
  compact: { minHeight: 40, paddingHorizontal: spacing.lg },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm }
});
