import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type FormFieldProps = {
  children: ReactNode;
  containerStyle?: StyleProp<ViewStyle>;
  error?: string | null;
  helperText?: string;
  label?: string;
  required?: boolean;
  /** Right-aligned text next to the label, e.g. a character counter. */
  hint?: string;
};

const makeStyles = (t: Theme) => ({
  container: { width: "100%" as const, gap: t.spacing.xs },
  labelRow: {
    flexDirection: "row" as const,
    alignItems: "center" as const,
    justifyContent: "space-between" as const,
    gap: t.spacing.xs
  }
});

export function FormField({
  children,
  containerStyle,
  error,
  helperText,
  hint,
  label,
  required = false
}: FormFieldProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={[styles.container, containerStyle]}>
      {label || hint ? (
        <View style={styles.labelRow}>
          {label ? (
            <Text tone="secondary" variant="label" weight="semibold">
              {required ? `${label} *` : label}
            </Text>
          ) : (
            <View />
          )}
          {hint ? (
            <Text tone="muted" variant="caption">
              {hint}
            </Text>
          ) : null}
        </View>
      ) : null}
      {children}
      {error ? (
        <Text accessibilityLiveRegion="polite" accessibilityRole="alert" tone="danger" variant="caption" weight="medium">
          {error}
        </Text>
      ) : helperText ? (
        <Text tone="muted" variant="caption">
          {helperText}
        </Text>
      ) : null}
    </View>
  );
}
