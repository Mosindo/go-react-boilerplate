import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { type Theme } from "./theme";
import { useThemedStyles } from "./useThemedStyles";

export type ProgressBarProps = {
  /** 0..1 */
  value: number;
  label?: string;
  style?: StyleProp<ViewStyle>;
};

const makeStyles = (t: Theme) => ({
  track: {
    height: 6,
    borderRadius: 3,
    backgroundColor: t.colors.surfaceMuted,
    overflow: "hidden" as const
  },
  fill: { height: 6, borderRadius: 3, backgroundColor: t.colors.primary }
});

export function ProgressBar({ label, style, value }: ProgressBarProps) {
  const styles = useThemedStyles(makeStyles);
  const clamped = Math.min(1, Math.max(0, value));
  return (
    <View
      accessibilityLabel={label ?? "Progress"}
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: 100, now: Math.round(clamped * 100) }}
      style={[styles.track, style]}
    >
      <View style={[styles.fill, { width: `${clamped * 100}%` }]} />
    </View>
  );
}
