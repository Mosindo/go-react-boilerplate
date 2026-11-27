import React, { useEffect, useRef } from "react";
import { ActivityIndicator, Animated, StyleSheet, View, type DimensionValue, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/palette";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityLabel={label ?? "Chargement"} accessible>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text tone="muted" variant="caption">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyStateProps = {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon, title, message, actionLabel, onAction, testID }: EmptyStateProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Icon name={icon} size={30} color={colors.primary} />
      </View>
      <Text variant="heading" align="center">
        {title}
      </Text>
      {message ? (
        <Text tone="muted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} variant="secondary" onPress={onAction} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

export function InlineError({ message }: { message?: string | null }) {
  const { colors } = useTheme();
  if (!message) {
    return null;
  }
  return (
    <View style={[styles.inline, { backgroundColor: colors.dangerSoft }]} accessibilityLiveRegion="polite" accessibilityRole="alert">
      <Icon name="alert-circle-outline" size={18} color={colors.danger} />
      <Text variant="caption" tone="danger" style={styles.flex}>
        {message}
      </Text>
    </View>
  );
}

export function Skeleton({ width = "100%", height, radius = radii.md, style }: { width?: DimensionValue; height: number; radius?: number; style?: StyleProp<ViewStyle> }) {
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
  return <Animated.View style={[{ width, height, borderRadius: radius, backgroundColor: colors.surfaceMuted, opacity }, style]} />;
}

export function ProgressBar({ value }: { value: number }) {
  const { colors } = useTheme();
  return (
    <View
      style={[styles.track, { backgroundColor: colors.surfaceMuted }]}
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}
    >
      <View style={[styles.fill, { width: `${Math.max(0, Math.min(1, value)) * 100}%`, backgroundColor: colors.primary }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl },
  iconWrap: { width: 68, height: 68, borderRadius: 34, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  message: { maxWidth: 320, marginBottom: spacing.xs },
  inline: { flexDirection: "row", gap: spacing.xs, alignItems: "center", padding: spacing.sm, borderRadius: radii.md },
  track: { height: 6, borderRadius: 3, overflow: "hidden" },
  fill: { height: 6, borderRadius: 3 }
});
