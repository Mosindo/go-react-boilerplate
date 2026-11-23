import React from "react";
import { StyleSheet, View } from "react-native";
import { Loader, Text, spacing } from "../ui";

export function LoadingView({ label }: { label?: string }) {
  return (
    <View style={styles.wrap} testID="loading-view">
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
  wrap: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.md
  }
});
