import React from "react";
import { StyleSheet, View } from "react-native";
import { Text, colors, spacing } from "../ui";

/** Wordmark: a soft glowing dot (the "lueur") next to the name. */
export function Logo({ size = 34 }: { size?: number }) {
  return (
    <View accessibilityLabel="Lueur" accessibilityRole="image" style={styles.row}>
      <View style={[styles.glow, { width: size * 0.62, height: size * 0.62, borderRadius: size }]}>
        <View style={[styles.core, { width: size * 0.3, height: size * 0.3, borderRadius: size }]} />
      </View>
      <Text style={{ fontSize: size, lineHeight: size * 1.15, letterSpacing: -1 }} variant="display">
        lueur
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs
  },
  glow: {
    backgroundColor: colors.accentSoft,
    alignItems: "center",
    justifyContent: "center"
  },
  core: {
    backgroundColor: colors.accent
  }
});
