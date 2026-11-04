import React, { useEffect, useRef } from "react";
import { ActivityIndicator, Animated, StyleSheet, View, type DimensionValue, type StyleProp, type ViewStyle } from "react-native";
import { Button } from "./Button";
import { Text } from "./Text";
import { radius, spacing, useColors } from "./theme";

export function Loader({ label }: { label?: string }) {
  const c = useColors();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={c.primary} size="large" />
      {label ? <Text tone="muted">{label}</Text> : null}
    </View>
  );
}

export function EmptyState({ title, message, actionLabel, onAction, testID }: { title: string; message?: string; actionLabel?: string; onAction?: () => void; testID?: string }) {
  return (
    <View style={styles.center} testID={testID}>
      <Text variant="heading" style={styles.centerText}>{title}</Text>
      {message ? <Text tone="muted" style={styles.centerText}>{message}</Text> : null}
      {actionLabel && onAction ? <Button title={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityLiveRegion="polite">
      <Text variant="heading" style={styles.centerText}>Oups</Text>
      <Text tone="muted" style={styles.centerText}>{message}</Text>
      {onRetry ? <Button title="Réessayer" onPress={onRetry} variant="secondary" /> : null}
    </View>
  );
}

export function Skeleton({ width = "100%", height = 16, style }: { width?: DimensionValue; height?: number; style?: StyleProp<ViewStyle> }) {
  const c = useColors();
  const opacity = useRef(new Animated.Value(0.5)).current;
  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 1, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.5, duration: 700, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);
  return <Animated.View accessibilityElementsHidden importantForAccessibility="no-hide-descendants" style={[{ width, height, borderRadius: radius.sm, backgroundColor: c.surfaceMuted, opacity }, style]} />;
}

export function ListSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <View style={styles.skeletonList} accessibilityLabel="Chargement">
      {Array.from({ length: rows }, (_, i) => (
        <View key={i} style={styles.skeletonRow}>
          <Skeleton width={52} height={52} style={{ borderRadius: 26 }} />
          <View style={{ flex: 1, gap: spacing.sm }}>
            <Skeleton width="50%" height={16} />
            <Skeleton width="80%" height={12} />
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  centerText: { textAlign: "center" },
  skeletonList: { padding: spacing.lg, gap: spacing.lg },
  skeletonRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
});
