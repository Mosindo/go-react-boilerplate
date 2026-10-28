import React, { useEffect, useRef } from "react";
import { ActivityIndicator, Animated, StyleSheet, View, type DimensionValue } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction,
  secondaryLabel,
  onSecondary,
  testID
}: {
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
  testID?: string;
}) {
  return (
    <View style={styles.center} testID={testID}>
      <Text variant="heading" style={styles.centerText}>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" style={styles.centerText}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button title={actionLabel} onPress={onAction} /> : null}
      {secondaryLabel && onSecondary ? <Button title={secondaryLabel} variant="ghost" onPress={onSecondary} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Text variant="heading" style={styles.centerText}>
        Something went wrong
      </Text>
      <Text tone="muted" style={styles.centerText}>
        {message}
      </Text>
      {onRetry ? <Button title="Try again" variant="secondary" onPress={onRetry} testID="retry-button" /> : null}
    </View>
  );
}

export function LoadingState({ label = "Loading…" }: { label?: string }) {
  return (
    <View style={styles.center} accessibilityLabel={label}>
      <ActivityIndicator />
    </View>
  );
}

export function Skeleton({ width = "100%", height = 16, radius = radii.sm }: { width?: DimensionValue; height?: number; radius?: number }) {
  const { colors } = useTheme();
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
  return <Animated.View style={{ width, height, borderRadius: radius, backgroundColor: colors.skeleton, opacity }} />;
}

export function ListSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <View style={{ gap: spacing.lg, padding: spacing.lg }} accessibilityLabel="Loading">
      {Array.from({ length: rows }, (_, i) => (
        <View key={i} style={{ flexDirection: "row", gap: spacing.md, alignItems: "center" }}>
          <Skeleton width={56} height={56} radius={28} />
          <View style={{ flex: 1, gap: spacing.sm }}>
            <Skeleton width="45%" />
            <Skeleton width="80%" height={12} />
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  centerText: { textAlign: "center" }
});
