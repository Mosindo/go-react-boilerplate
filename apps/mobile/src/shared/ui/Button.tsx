import React, { type ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, View, type PressableProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { controls, radii, spacing } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  label: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  fullWidth?: boolean;
  leading?: ReactNode;
  style?: StyleProp<ViewStyle>;
};

export function Button({
  label,
  variant = "primary",
  size = "md",
  loading = false,
  fullWidth = true,
  leading,
  disabled,
  style,
  ...props
}: ButtonProps) {
  const { colors } = useTheme();
  const inactive = Boolean(disabled) || loading;

  const palette = {
    primary: { bg: colors.primary, fg: colors.primaryForeground, border: colors.primary },
    secondary: { bg: colors.primarySoft, fg: colors.primary, border: colors.primarySoft },
    outline: { bg: "transparent", fg: colors.text, border: colors.borderStrong },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    destructive: { bg: colors.danger, fg: "#FFFFFF", border: colors.danger }
  }[variant];

  return (
    <Pressable
      accessibilityLabel={props.accessibilityLabel ?? label}
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      style={({ pressed }) => [
        styles.base,
        {
          minHeight: controls.button[size],
          backgroundColor: palette.bg,
          borderColor: palette.border,
          opacity: inactive ? 0.55 : pressed ? 0.85 : 1,
          alignSelf: fullWidth ? "stretch" : "flex-start"
        },
        style
      ]}
      {...props}
    >
      <View style={styles.row}>
        {loading ? <ActivityIndicator color={palette.fg} size="small" /> : leading}
        <Text style={{ color: palette.fg }} variant="button">
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.xl
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  }
});
