import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Loader, Text, spacing } from "../ui";

type LoadingViewProps = {
  label?: string;
  style?: StyleProp<ViewStyle>;
};

export function LoadingView({ label, style }: LoadingViewProps) {
  return (
    <View style={[styles.root, style]} testID="loading-view">
      <Loader size="large" />
      {label ? <Text tone="muted">{label}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({ root: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md } });
