import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, spacing } from "./tokens";

export type FormFieldProps = {
  children: ReactNode;
  containerStyle?: StyleProp<ViewStyle>;
  error?: string | null;
  helperText?: string;
  label?: string;
  required?: boolean;
};

export function FormField({
  children,
  containerStyle,
  error,
  helperText,
  label,
  required = false
}: FormFieldProps) {
  return (
    <View style={[styles.container, containerStyle]}>
      {label ? (
        <View style={styles.labelRow}>
          <Text tone="secondary" variant="label" weight="semibold">
            {label}
          </Text>
          {required ? (
            <Text style={styles.required} tone="danger" variant="label" weight="bold">
              *
            </Text>
          ) : null}
        </View>
      ) : null}
      {children}
      {error ? (
        <Text tone="danger" variant="caption" weight="medium">
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

const styles = StyleSheet.create({
  container: {
    width: "100%",
    gap: spacing.sm
  },
  labelRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  required: {
    color: colors.danger
  }
});
