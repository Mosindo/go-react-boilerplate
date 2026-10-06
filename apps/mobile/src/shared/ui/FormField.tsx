import React, { type ReactNode } from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { spacing } from "./tokens";

export type FormFieldProps = {
  label?: string;
  helperText?: string;
  error?: string | null;
  children: ReactNode;
  containerStyle?: StyleProp<ViewStyle>;
};

export function FormField({ label, helperText, error, children, containerStyle }: FormFieldProps) {
  return (
    <View style={[styles.root, containerStyle]}>
      {label ? (
        <Text tone="secondary" variant="label" weight="semibold">
          {label}
        </Text>
      ) : null}
      {children}
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
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
  root: { gap: spacing.xs }
});
