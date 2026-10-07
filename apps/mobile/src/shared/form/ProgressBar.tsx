import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme/useTheme";
import { radii } from "../ui";

export function ProgressBar({ current, total }: { current: number; total: number }) {
  const { colors } = useTheme();
  const ratio = total > 0 ? Math.min(1, Math.max(0, current / total)) : 0;
  return (
    <View
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: total, now: current }}
      style={[styles.track, { backgroundColor: colors.surfaceSubtle }]}
    >
      <View style={[styles.fill, { width: `${ratio * 100}%`, backgroundColor: colors.primary }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  track: { height: 6, borderRadius: radii.pill, overflow: "hidden", width: "100%" },
  fill: { height: "100%", borderRadius: radii.pill }
});
