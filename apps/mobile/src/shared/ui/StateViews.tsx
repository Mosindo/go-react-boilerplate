import React, { useEffect, useRef } from 'react';
import {
  ActivityIndicator,
  Animated,
  StyleSheet,
  View,
  type DimensionValue,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { fr } from '../../i18n/fr';
import { radius, spacing, useTheme } from '../../theme';
import { Button } from './Button';
import { Text } from './Text';

export function LoadingView({ label, testID }: { label?: string; testID?: string }) {
  const { colors } = useTheme();
  return (
    <View
      style={styles.center}
      testID={testID ?? 'loading'}
      accessibilityRole="progressbar"
      accessibilityLabel={label ?? fr.common.loading}
    >
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? (
        <Text variant="caption" tone="muted" style={styles.gap}>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

export type EmptyStateProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  body?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({
  icon = 'sparkles-outline',
  title,
  body,
  actionLabel,
  onAction,
  testID,
}: EmptyStateProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={32} color={colors.primary} />
      </View>
      <Text variant="title" center style={styles.gap}>
        {title}
      </Text>
      {body ? (
        <Text tone="muted" center style={styles.gapSm}>
          {body}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <Button label={actionLabel} onPress={onAction} style={styles.action} variant="secondary" />
      ) : null}
    </View>
  );
}

export type ErrorStateProps = {
  title?: string;
  message?: string;
  onRetry?: () => void;
  testID?: string;
};

export function ErrorState({ title, message, onRetry, testID }: ErrorStateProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID ?? 'error-state'} accessibilityRole="alert">
      <View style={[styles.iconWrap, { backgroundColor: colors.dangerSoft }]}>
        <Ionicons name="cloud-offline-outline" size={32} color={colors.danger} />
      </View>
      <Text variant="heading" center style={styles.gap}>
        {title ?? fr.errors.loadFailed}
      </Text>
      {message ? (
        <Text tone="muted" center style={styles.gapSm}>
          {message}
        </Text>
      ) : null}
      {onRetry ? (
        <Button
          label={fr.common.retry}
          icon="refresh"
          onPress={onRetry}
          style={styles.action}
          testID="retry"
        />
      ) : null}
    </View>
  );
}

export function Skeleton({
  width = '100%',
  height = 16,
  rounded = radius.sm,
  style,
}: {
  width?: DimensionValue;
  height?: number;
  rounded?: number;
  style?: StyleProp<ViewStyle>;
}) {
  const { colors } = useTheme();
  const opacity = useRef(new Animated.Value(0.55)).current;
  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 1, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.55, duration: 700, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);
  return (
    <Animated.View
      accessible={false}
      style={[
        { width, height, borderRadius: rounded, backgroundColor: colors.skeleton, opacity },
        style,
      ]}
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: spacing.xl },
  iconWrap: {
    width: 72,
    height: 72,
    borderRadius: 36,
    alignItems: 'center',
    justifyContent: 'center',
  },
  gap: { marginTop: spacing.lg },
  gapSm: { marginTop: spacing.sm },
  action: { marginTop: spacing.xl },
});
