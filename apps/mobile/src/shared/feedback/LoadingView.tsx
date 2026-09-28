import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Loader } from "../ui/Loader";
import { Text } from "../ui/Text";

export function LoadingView({ label = "Loading" }: { label?: string }) {
  return (
    <View accessibilityLabel={label} accessibilityLiveRegion="polite" style={styles.wrap}>
      <Loader size="large" />
      <Text tone="muted">{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
