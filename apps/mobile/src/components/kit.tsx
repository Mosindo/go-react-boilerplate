import React, { useEffect, useRef, useState, type ReactNode } from "react";
import {
  AccessibilityInfo,
  ActivityIndicator,
  Animated,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text as RNText,
  View,
  type StyleProp,
  type TextProps as RNTextProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { lightColors, useTheme, type Colors } from "../shared/ui/theme";

/** Text drawn over photos is always white, in both themes. */
export const photoTextColor = lightColors.surface;

/** Minimum touch target (points). */
export const MIN_TARGET = 44;

export function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    let mounted = true;
    void AccessibilityInfo.isReduceMotionEnabled()
      .then((value) => {
        if (mounted) setReduced(value);
      })
      .catch(() => undefined);
    const sub = AccessibilityInfo.addEventListener("reduceMotionChanged", setReduced);
    return () => {
      mounted = false;
      sub.remove();
    };
  }, []);
  return reduced;
}

export type TxtVariant = "title" | "heading" | "subheading" | "body" | "label" | "caption";
export type TxtTone = "default" | "muted" | "subtle" | "primary" | "danger" | "inverse" | "success";

type TxtProps = RNTextProps & {
  variant?: TxtVariant;
  tone?: TxtTone;
  weight?: "regular" | "medium" | "semibold" | "bold";
  style?: StyleProp<TextStyle>;
};

export function toneColor(colors: Colors, tone: TxtTone): string {
  switch (tone) {
    case "muted":
      return colors.textMuted;
    case "subtle":
      return colors.textSubtle;
    case "primary":
      return colors.primary;
    case "danger":
      return colors.danger;
    case "success":
      return colors.success;
    case "inverse":
      return photoTextColor;
    default:
      return colors.text;
  }
}

export function Txt({ variant = "body", tone = "default", weight, style, ...rest }: TxtProps) {
  const { colors, fontWeights } = useTheme();
  const size: Record<TxtVariant, TextStyle> = {
    title: { fontSize: 30, lineHeight: 36, letterSpacing: -0.6 },
    heading: { fontSize: 22, lineHeight: 28, letterSpacing: -0.3 },
    subheading: { fontSize: 17, lineHeight: 22 },
    body: { fontSize: 15, lineHeight: 21 },
    label: { fontSize: 13, lineHeight: 18 },
    caption: { fontSize: 12, lineHeight: 16 }
  };
  const defaultWeight = variant === "title" || variant === "heading" ? "bold" : "regular";
  return (
    <RNText
      {...rest}
      style={[
        size[variant],
        {
          color: toneColor(colors, tone),
          fontWeight: fontWeights[weight ?? defaultWeight]
        },
        style
      ]}
    />
  );
}

export type AppButtonVariant = "filled" | "soft" | "outline" | "ghost" | "danger";

type AppButtonProps = {
  label: string;
  onPress: () => void;
  variant?: AppButtonVariant;
  loading?: boolean;
  disabled?: boolean;
  fullWidth?: boolean;
  testID?: string;
  accessibilityHint?: string;
  style?: StyleProp<ViewStyle>;
};

export function AppButton({
  label,
  onPress,
  variant = "filled",
  loading = false,
  disabled = false,
  fullWidth = false,
  testID,
  accessibilityHint,
  style
}: AppButtonProps) {
  const { colors, radii, spacing } = useTheme();
  const inactive = disabled || loading;
  const palette: Record<AppButtonVariant, { bg: string; fg: string; border: string }> = {
    filled: {
      bg: colors.primary,
      fg: colors.primaryForeground,
      border: colors.primary
    },
    soft: {
      bg: colors.primarySoft,
      fg: colors.primary,
      border: colors.primarySoft
    },
    outline: {
      bg: "transparent",
      fg: colors.text,
      border: colors.borderStrong
    },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    danger: {
      bg: colors.dangerSoft,
      fg: colors.danger,
      border: colors.dangerSoft
    }
  };
  const p = palette[variant];
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityHint={accessibilityHint}
      accessibilityState={{ disabled: inactive, busy: loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.button,
        {
          minHeight: 48,
          minWidth: MIN_TARGET,
          borderRadius: radii.pill,
          paddingHorizontal: spacing.xl,
          backgroundColor: p.bg,
          borderColor: p.border,
          opacity: inactive ? 0.5 : pressed ? 0.85 : 1
        },
        fullWidth ? styles.fullWidth : null,
        style
      ]}
    >
      {loading ? <ActivityIndicator size="small" color={p.fg} /> : null}
      <Txt variant="subheading" weight="semibold" style={{ color: p.fg }}>
        {label}
      </Txt>
    </Pressable>
  );
}

type IconButtonProps = {
  glyph: string;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  backgroundColor?: string;
  borderColor?: string;
  glyphSize?: number;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

/** Round glyph button with an accessible label and at least a 44pt hit area. */
export function IconButton({
  glyph,
  label,
  onPress,
  size = MIN_TARGET,
  color,
  backgroundColor,
  borderColor,
  glyphSize = 20,
  disabled,
  testID,
  style
}: IconButtonProps) {
  const { colors } = useTheme();
  const dim = Math.max(size, MIN_TARGET);
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: Boolean(disabled) }}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      hitSlop={4}
      style={({ pressed }) => [
        styles.iconButton,
        {
          width: dim,
          height: dim,
          borderRadius: dim / 2,
          backgroundColor: backgroundColor ?? "transparent",
          borderColor: borderColor ?? "transparent",
          opacity: disabled ? 0.4 : pressed ? 0.75 : 1
        },
        style
      ]}
    >
      <RNText
        importantForAccessibility="no"
        style={{
          color: color ?? colors.text,
          fontSize: glyphSize,
          fontWeight: "700"
        }}
      >
        {glyph}
      </RNText>
    </Pressable>
  );
}

export function Chip({ label, onPhoto = false }: { label: string; onPhoto?: boolean }) {
  const { colors, radii, spacing } = useTheme();
  return (
    <View
      style={{
        paddingHorizontal: spacing.sm + 2,
        paddingVertical: spacing.xxs + 1,
        borderRadius: radii.pill,
        backgroundColor: onPhoto ? colors.photoScrim : colors.primarySoft,
        borderWidth: onPhoto ? StyleSheet.hairlineWidth : 0,
        borderColor: photoTextColor
      }}
    >
      <Txt variant="caption" weight="semibold" tone={onPhoto ? "inverse" : "primary"}>
        {label}
      </Txt>
    </View>
  );
}

/** Pulsing placeholder block for loading states. */
export function Skeleton({ style }: { style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  const reduced = useReducedMotion();
  const pulse = useRef(new Animated.Value(0.55)).current;
  useEffect(() => {
    if (reduced) {
      pulse.setValue(0.7);
      return undefined;
    }
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(pulse, {
          toValue: 1,
          duration: 800,
          useNativeDriver: true
        }),
        Animated.timing(pulse, {
          toValue: 0.55,
          duration: 800,
          useNativeDriver: true
        })
      ])
    );
    loop.start();
    return () => loop.stop();
  }, [pulse, reduced]);
  return (
    <Animated.View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={[
        {
          backgroundColor: colors.surfaceMuted,
          opacity: pulse,
          borderRadius: 8
        },
        style
      ]}
    />
  );
}

export type BannerTone = "info" | "danger" | "success" | "warning";

type BannerProps = {
  message: string;
  tone?: BannerTone;
  actionLabel?: string;
  onAction?: () => void;
  onDismiss?: () => void;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

/** Inline, non-blocking notice. */
export function Banner({
  message,
  tone = "info",
  actionLabel,
  onAction,
  onDismiss,
  testID,
  style
}: BannerProps) {
  const { colors, radii, spacing } = useTheme();
  const palette: Record<BannerTone, { bg: string; fg: TxtTone }> = {
    info: { bg: colors.surfaceMuted, fg: "default" },
    danger: { bg: colors.dangerSoft, fg: "danger" },
    success: { bg: colors.successSoft, fg: "success" },
    warning: { bg: colors.warningSoft, fg: "default" }
  };
  const p = palette[tone];
  return (
    <View
      accessibilityRole="alert"
      accessibilityLiveRegion="polite"
      testID={testID}
      style={[
        styles.banner,
        {
          backgroundColor: p.bg,
          borderRadius: radii.md,
          padding: spacing.md,
          gap: spacing.sm
        },
        style
      ]}
    >
      <Txt variant="label" tone={p.fg} weight="semibold" style={styles.flex}>
        {message}
      </Txt>
      {actionLabel && onAction ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={actionLabel}
          onPress={onAction}
          hitSlop={8}
          style={styles.bannerAction}
        >
          <Txt variant="label" tone="primary" weight="bold">
            {actionLabel}
          </Txt>
        </Pressable>
      ) : null}
      {onDismiss ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Dismiss"
          onPress={onDismiss}
          hitSlop={8}
          style={styles.bannerAction}
        >
          <Txt variant="label" tone="muted" weight="bold">
            Dismiss
          </Txt>
        </Pressable>
      ) : null}
    </View>
  );
}

type StateProps = {
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  refreshing?: boolean;
  onRefresh?: () => void;
  testID?: string;
  glyph?: string;
  children?: ReactNode;
};

/** Centered explanatory state (empty or error) that also supports pull-to-refresh. */
export function StateView({
  title,
  message,
  actionLabel,
  onAction,
  refreshing,
  onRefresh,
  testID,
  glyph,
  children
}: StateProps) {
  const { colors, spacing } = useTheme();
  return (
    <ScrollView
      testID={testID}
      contentContainerStyle={[styles.stateContent, { padding: spacing.xl, gap: spacing.md }]}
      refreshControl={
        onRefresh ? (
          <RefreshControl
            refreshing={Boolean(refreshing)}
            onRefresh={onRefresh}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        ) : undefined
      }
    >
      {glyph ? (
        <RNText importantForAccessibility="no" style={styles.stateGlyph}>
          {glyph}
        </RNText>
      ) : null}
      <Txt variant="heading" style={styles.center} accessibilityRole="header">
        {title}
      </Txt>
      {message ? (
        <Txt tone="muted" style={styles.center}>
          {message}
        </Txt>
      ) : null}
      {children}
      {actionLabel && onAction ? (
        <AppButton label={actionLabel} onPress={onAction} variant="soft" />
      ) : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  button: {
    alignItems: "center",
    justifyContent: "center",
    flexDirection: "row",
    gap: 8,
    borderWidth: 1
  },
  fullWidth: { alignSelf: "stretch" },
  iconButton: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1
  },
  banner: { flexDirection: "row", alignItems: "center" },
  bannerAction: {
    minHeight: MIN_TARGET,
    justifyContent: "center",
    paddingHorizontal: 4
  },
  flex: { flex: 1 },
  center: { textAlign: "center" },
  stateContent: { flexGrow: 1, alignItems: "center", justifyContent: "center" },
  stateGlyph: { fontSize: 44 }
});
