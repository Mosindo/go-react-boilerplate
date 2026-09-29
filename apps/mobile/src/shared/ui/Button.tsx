import React, { type ReactNode } from "react";
import {
  ActivityIndicator,
  Pressable,
  View,
  type PressableProps,
  type StyleProp,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { Text, type TextTone } from "./Text";
import { useTheme, type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";
import { controls } from "./tokens";

export type ButtonVariant = "primary" | "secondary" | "outline" | "ghost" | "destructive" | "success";
export type ButtonSize = "sm" | "md" | "lg";

export type ButtonProps = Omit<PressableProps, "children" | "style"> & {
  children?: ReactNode;
  label?: string;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
  textStyle?: StyleProp<TextStyle>;
};

const makeStyles = (t: Theme) => ({
  base: {
    borderRadius: t.radii.lg,
    alignItems: "center" as const,
    justifyContent: "center" as const
  },
  fullWidth: { width: "100%" as const },
  sm: { minHeight: controls.button.sm, paddingHorizontal: t.spacing.lg, paddingVertical: t.spacing.xs },
  md: { minHeight: controls.button.md, paddingHorizontal: t.spacing.xl, paddingVertical: t.spacing.sm },
  lg: { minHeight: controls.button.lg, paddingHorizontal: t.spacing.xl, paddingVertical: t.spacing.md },
  primary: { backgroundColor: t.colors.primary },
  secondary: { backgroundColor: t.colors.primarySoft, borderWidth: 1, borderColor: t.colors.primary },
  destructive: { backgroundColor: t.colors.danger },
  success: { backgroundColor: t.colors.success },
  outline: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.borderStrong },
  ghost: { backgroundColor: "transparent" },
  disabled: { opacity: 0.5 },
  pressed: { opacity: 0.85 },
  content: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "center" as const,
    gap: t.spacing.sm
  }
});

const textToneByVariant: Record<ButtonVariant, TextTone> = {
  primary: "inverse",
  secondary: "primary",
  destructive: "inverse",
  success: "inverse",
  outline: "default",
  ghost: "primary"
};

export function Button({
  accessibilityLabel,
  accessibilityState,
  children,
  disabled,
  fullWidth = false,
  label,
  loading = false,
  size = "md",
  style,
  textStyle,
  variant = "primary",
  ...props
}: ButtonProps) {
  const theme = useTheme();
  const styles = useThemedStyles(makeStyles);
  const buttonDisabled = Boolean(disabled) || loading;
  const textTone = textToneByVariant[variant];
  const labelNode = typeof children === "string" ? children : label;
  const spinnerColor =
    textTone === "inverse" ? theme.colors.primaryForeground : theme.colors.primary;

  return (
    <Pressable
      accessibilityLabel={accessibilityLabel ?? labelNode}
      accessibilityRole="button"
      accessibilityState={{ ...accessibilityState, disabled: buttonDisabled, busy: loading }}
      disabled={buttonDisabled}
      style={({ pressed }) => [
        styles.base,
        styles[size],
        styles[variant],
        fullWidth ? styles.fullWidth : null,
        buttonDisabled ? styles.disabled : null,
        pressed ? styles.pressed : null,
        style
      ]}
      {...props}
    >
      <View style={styles.content}>
        {loading ? <ActivityIndicator color={spinnerColor} size="small" /> : null}
        {labelNode ? (
          <Text style={textStyle} tone={textTone} variant="button" weight="bold">
            {labelNode}
          </Text>
        ) : (
          children
        )}
      </View>
    </Pressable>
  );
}
