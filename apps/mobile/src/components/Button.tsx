import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, radii, spacing } from "../theme/tokens";
import { AppText } from "./AppText";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = {
  label: string;
  onPress: () => void;
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
  primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary },
  secondary: { bg: colors.surfaceRaised, pressed: colors.surfacePressed, fg: colors.text, border: colors.border },
  ghost: { bg: "transparent", pressed: colors.surfaceRaised, fg: colors.text, border: "transparent" },
  danger: { bg: colors.dangerSoft, pressed: colors.surfacePressed, fg: colors.danger, border: colors.danger }
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
        { backgroundColor: pressed ? p.pressed : p.bg, borderColor: p.border },
        inactive && styles.disabled,
        style
      ]}
    >
      <View style={styles.content}>
        {loading ? (
          <ActivityIndicator color={p.fg} />
        ) : (
          <>
            {icon ? <Ionicons name={icon} size={18} color={p.fg} /> : null}
            <AppText variant="bodyStrong" color={p.fg}>
              {label}
            </AppText>
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
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  compact: { minHeight: 40, paddingHorizontal: spacing.md },
  content: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  disabled: { opacity: 0.5 }
});
