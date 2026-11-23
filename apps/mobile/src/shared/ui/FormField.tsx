import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { spacing } from "./tokens";

type FormFieldProps = {
  children: ReactNode;
  label?: string;
  helperText?: string;
  error?: string | null;
  style?: StyleProp<ViewStyle>;
};

export function FormField({ children, error, helperText, label, style }: FormFieldProps) {
  return (
    <View style={[styles.wrap, style]}>
      {label ? (
        <Text tone="muted" variant="label" weight="medium">
          {label}
        </Text>
      ) : null}
      {children}
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : helperText ? (
        <Text tone="subtle" variant="caption">
          {helperText}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.xs
  }
});
