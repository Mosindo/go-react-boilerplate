import React from "react";
import { ActivityIndicator, StyleSheet, View, type ActivityIndicatorProps, type StyleProp, type ViewStyle } from "react-native";
import { Text } from "./Text";
import { colors, spacing, typography } from "./tokens";

export type LoaderProps = {
  fullScreen?: boolean;
  label?: string;
  size?: ActivityIndicatorProps["size"];
  style?: StyleProp<ViewStyle>;
};

const styles = StyleSheet.create({
  inline: {
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm
  },
  fullScreen: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.md
  }
});

export function Loader({ fullScreen = false, label, size = "large", style }: LoaderProps) {
  return (
    <View style={[fullScreen ? styles.fullScreen : styles.inline, style]}>
      <ActivityIndicator color={colors.primary} size={size} />
      {label ? (
        <Text style={typography.label} tone="muted">
          {label}
        </Text>
      ) : null}
    </View>
  );
}
