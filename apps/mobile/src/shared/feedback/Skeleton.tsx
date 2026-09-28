import React, { useEffect, useRef } from "react";
import { Animated, StyleSheet, View, type DimensionValue, type StyleProp, type ViewStyle } from "react-native";
import { radius, spacing, useTheme } from "../../theme";

/** Pulsing placeholder block. */
export function Skeleton({
  width = "100%",
  height = 16,
  rounded = radius.sm,
  style
}: {
  width?: DimensionValue;
  height?: number;
  rounded?: number;
  style?: StyleProp<ViewStyle>;
}) {
  const theme = useTheme();
  const opacity = useRef(new Animated.Value(0.5)).current;
  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 1, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.5, duration: 700, useNativeDriver: true })
      ])
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);
  return (
    <Animated.View
      style={[{ width, height, borderRadius: rounded, backgroundColor: theme.skeleton, opacity }, style]}
    />
  );
}

/** Placeholder rows for list screens. */
export function ListSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <View accessibilityLabel="Loading" style={styles.list}>
      {Array.from({ length: rows }, (_, index) => (
        <View key={index} style={styles.row}>
          <Skeleton height={48} rounded={24} width={48} />
          <View style={styles.lines}>
            <Skeleton height={14} width="45%" />
            <Skeleton height={12} width="80%" />
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.lg, padding: spacing.lg },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  lines: { flex: 1, gap: spacing.sm }
});
