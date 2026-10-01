import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Loader, Text, spacing } from "../ui";

type LoadingViewProps = { label?: string; fullScreen?: boolean; style?: StyleProp<ViewStyle> };

export function LoadingView({ label, fullScreen = false, style }: LoadingViewProps) {
  return (
    <View accessibilityLiveRegion="polite" style={[styles.root, fullScreen && styles.full, style]}>
      <Loader size="large" />
      {label ? (
        <Text tone="muted" variant="label">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  full: { flex: 1 }
});
