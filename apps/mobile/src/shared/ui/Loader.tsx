import React from "react";
import { ActivityIndicator, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { spacing } from "./tokens";

export type LoaderProps = {
  label?: string;
  size?: "small" | "large";
  style?: StyleProp<ViewStyle>;
};

export function Loader({ label, size = "large", style }: LoaderProps) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={[styles.root, style]}>
      <ActivityIndicator color={colors.primary} size={size} />
      {label ? (
        <Text tone="muted" variant="label">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.lg }
});
