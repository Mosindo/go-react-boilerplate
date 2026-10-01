import React from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text as RNText,
  TextInput,
  View,
  type StyleProp,
  type TextInputProps,
  type TextProps,
  type TextStyle,
  type ViewStyle
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { MIN_TOUCH, radii, spacing, useTheme } from "../theme/theme";

type Variant = "display" | "title" | "heading" | "body" | "label" | "caption";

const variants: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800", letterSpacing: -0.8 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.4 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 23, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 20, fontWeight: "600" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" }
};

export function Text({
  variant = "body",
  muted,
  color,
  style,
  ...rest
}: TextProps & { variant?: Variant; muted?: boolean; color?: string }) {
  const t = useTheme();
  return <RNText {...rest} style={[variants[variant], { color: color ?? (muted ? t.textMuted : t.text) }, style]} />;
}

export function Screen({
  children,
  scroll = false,
  padded = true,
  style,
  edges = ["top", "left", "right"]
}: {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  style?: StyleProp<ViewStyle>;
  edges?: ("top" | "bottom" | "left" | "right")[];
}) {
  const t = useTheme();
  const inner = padded ? { paddingHorizontal: spacing.lg } : null;
  return (
    <SafeAreaView edges={edges} style={[{ flex: 1, backgroundColor: t.background }, style]}>
      {scroll ? (
        <ScrollView
          contentContainerStyle={[inner, { paddingBottom: spacing.xxl, flexGrow: 1 }]}
          keyboardShouldPersistTaps="handled"
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[{ flex: 1 }, inner]}>{children}</View>
      )}
    </SafeAreaView>
  );
}

type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({ label, onPress, variant = "primary", loading, disabled, icon, style, testID }: ButtonProps) {
  const t = useTheme();
  const palette = {
    primary: { bg: t.primary, fg: t.onPrimary, border: t.primary },
    secondary: { bg: t.surface, fg: t.text, border: t.border },
    ghost: { bg: "transparent", fg: t.primary, border: "transparent" },
    danger: { bg: t.dangerSoft, fg: t.danger, border: t.dangerSoft }
  }[variant];
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.button,
        { backgroundColor: palette.bg, borderColor: palette.border, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <>
          {icon ? <Ionicons name={icon} size={20} color={palette.fg} style={{ marginRight: spacing.sm }} /> : null}
          <RNText style={[variants.label, { color: palette.fg, fontSize: 16 }]}>{label}</RNText>
        </>
      )}
    </Pressable>
  );
}

export function IconButton({
  icon,
  label,
  onPress,
  color,
  size = 24,
  testID,
  style
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  color?: string;
  size?: number;
  testID?: string;
  style?: StyleProp<ViewStyle>;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.iconButton, { opacity: pressed ? 0.6 : 1 }, style]}
    >
      <Ionicons name={icon} size={size} color={color ?? t.text} />
    </Pressable>
  );
}

export function TextField({
  label,
  error,
  hint,
  style,
  ...rest
}: TextInputProps & { label: string; error?: string | null; hint?: string }) {
  const t = useTheme();
  return (
    <View style={{ marginBottom: spacing.lg }}>
      <Text variant="label" style={{ marginBottom: spacing.xs }}>
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={t.textMuted}
        {...rest}
        style={[
          styles.input,
          { backgroundColor: t.surface, color: t.text, borderColor: error ? t.danger : t.border },
          rest.multiline ? { minHeight: 110, textAlignVertical: "top", paddingTop: spacing.md } : null,
          style
        ]}
      />
      {error ? (
        <Text variant="caption" color={t.danger} style={{ marginTop: spacing.xs }} accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" muted style={{ marginTop: spacing.xs }}>
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

export function Chip({
  label,
  selected,
  onPress,
  testID
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole={onPress ? "checkbox" : "text"}
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      disabled={!onPress}
      onPress={onPress}
      testID={testID}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? t.primarySoft : t.surface,
          borderColor: selected ? t.primary : t.border
        }
      ]}
    >
      <RNText style={[variants.label, { color: selected ? t.primary : t.text }]}>{label}</RNText>
    </Pressable>
  );
}

export function ErrorText({ children, style }: { children: React.ReactNode; style?: StyleProp<TextStyle> }) {
  const t = useTheme();
  return (
    <Text variant="caption" color={t.danger} style={style} accessibilityLiveRegion="polite">
      {children}
    </Text>
  );
}

export function Row({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  return <View style={[{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }, style]}>{children}</View>;
}

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const t = useTheme();
  return (
    <View style={[styles.card, { backgroundColor: t.surface, borderColor: t.border }, style]}>{children}</View>
  );
}

export function Badge({ count }: { count: number }) {
  const t = useTheme();
  if (count <= 0) return null;
  return (
    <View style={[styles.badge, { backgroundColor: t.primary }]} accessibilityLabel={`${count} non lus`}>
      <RNText style={{ color: t.onPrimary, fontSize: 12, fontWeight: "700" }}>{count > 99 ? "99+" : count}</RNText>
    </View>
  );
}

export function Loading({ label = "Chargement…" }: { label?: string }) {
  const t = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator size="large" color={t.primary} />
    </View>
  );
}

export function EmptyState({
  icon,
  title,
  message,
  actionLabel,
  onAction
}: {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  const t = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.emptyIcon, { backgroundColor: t.accentSoft }]}>
        <Ionicons name={icon} size={36} color={t.accent} />
      </View>
      <Text variant="heading" style={{ textAlign: "center", marginBottom: spacing.sm }}>
        {title}
      </Text>
      <Text muted style={{ textAlign: "center", marginBottom: spacing.xl, maxWidth: 320 }}>
        {message}
      </Text>
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups, un souci"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

export function Separator() {
  const t = useTheme();
  return <View style={{ height: StyleSheet.hairlineWidth, backgroundColor: t.border }} />;
}

const styles = StyleSheet.create({
  button: {
    minHeight: 52,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center"
  },
  iconButton: { minWidth: MIN_TOUCH, minHeight: MIN_TOUCH, alignItems: "center", justifyContent: "center" },
  input: {
    minHeight: 52,
    borderRadius: radii.md,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    fontSize: 16
  },
  chip: {
    minHeight: 40,
    borderRadius: radii.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.lg,
    alignItems: "center",
    justifyContent: "center"
  },
  card: { borderRadius: radii.lg, borderWidth: 1, padding: spacing.lg },
  badge: {
    minWidth: 22,
    height: 22,
    paddingHorizontal: 6,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center"
  },
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  emptyIcon: {
    width: 80,
    height: 80,
    borderRadius: 40,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.lg
  }
});
