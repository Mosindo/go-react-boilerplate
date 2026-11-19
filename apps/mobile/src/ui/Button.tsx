import React from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/tokens";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type Variant = "primary" | "secondary" | "ghost" | "danger";

export type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: Variant;
  loading?: boolean;
  disabled?: boolean;
  icon?: IconName;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  accessibilityHint?: string;
};

export function Button({ label, onPress, variant = "primary", loading, disabled, icon, style, testID, accessibilityHint }: ButtonProps) {
  const { colors } = useTheme();
  const inactive = disabled || loading;
  const palette = {
    primary: { bg: colors.primary, pressed: colors.primaryPressed, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.surface, pressed: colors.surfaceMuted, fg: colors.text, border: colors.borderStrong },
    ghost: { bg: "transparent", pressed: colors.surfaceMuted, fg: colors.primary, border: "transparent" },
    danger: { bg: colors.dangerSoft, pressed: colors.dangerSoft, fg: colors.danger, border: colors.dangerSoft }
  }[variant];

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
        { backgroundColor: pressed ? palette.pressed : palette.bg, borderColor: palette.border, opacity: inactive ? 0.55 : 1 },
        style
      ]}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={palette.fg} size="small" /> : icon ? <Icon name={icon} size={19} color={palette.fg} /> : null}
        <Text variant="label" style={{ color: palette.fg, fontSize: 16 }}>
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
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs }
});
