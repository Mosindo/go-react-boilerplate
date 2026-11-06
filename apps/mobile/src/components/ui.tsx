import React, { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
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
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../theme";
import { fr } from "../i18n/fr";

type Variant = "display" | "title" | "heading" | "body" | "caption" | "label";

const sizes: Record<Variant, TextStyle> = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: "800", letterSpacing: -0.5 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: "700", letterSpacing: -0.3 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 22, fontWeight: "400" },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: "400" },
  label: { fontSize: 14, lineHeight: 18, fontWeight: "600" }
};

export function Text({
  variant = "body",
  muted,
  color,
  style,
  ...props
}: TextProps & { variant?: Variant; muted?: boolean; color?: string }) {
  const { colors } = useTheme();
  return (
    <RNText
      {...props}
      style={[sizes[variant], { color: color ?? (muted ? colors.textMuted : colors.text) }, style]}
    />
  );
}

type ButtonProps = {
  title: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  accessibilityLabel?: string;
};

export function Button({
  title,
  onPress,
  variant = "primary",
  loading,
  disabled,
  style,
  testID,
  accessibilityLabel
}: ButtonProps) {
  const { colors } = useTheme();
  const inactive = disabled || loading;
  const palette = {
    primary: { bg: colors.primary, fg: colors.primaryText, border: colors.primary },
    secondary: { bg: colors.surface, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    danger: { bg: colors.danger, fg: "#FFFFFF", border: colors.danger }
  }[variant];
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? title}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      style={({ pressed }) => [
        styles.button,
        {
          backgroundColor: palette.bg,
          borderColor: palette.border,
          opacity: inactive ? 0.55 : pressed ? 0.85 : 1
        },
        style
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <RNText style={[sizes.label, { fontSize: 16, color: palette.fg }]}>{title}</RNText>
      )}
    </Pressable>
  );
}

export function IconButton({
  label,
  glyph,
  onPress,
  size = 56,
  tone = "neutral",
  testID
}: {
  label: string;
  glyph: string;
  onPress: () => void;
  size?: number;
  tone?: "neutral" | "primary";
  testID?: string;
}) {
  const { colors } = useTheme();
  const primary = tone === "primary";
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.iconButton,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: primary ? colors.primary : colors.surface,
          borderColor: primary ? colors.primary : colors.border,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <RNText style={{ fontSize: size * 0.42, color: primary ? colors.primaryText : colors.text }}>
        {glyph}
      </RNText>
    </Pressable>
  );
}

type InputProps = Omit<TextInputProps, "style"> & {
  label: string;
  error?: string | null;
  hint?: string;
  testID?: string;
};

export function Input({
  label,
  error,
  hint,
  secureTextEntry,
  testID,
  multiline,
  ...props
}: InputProps) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  const [hidden, setHidden] = useState(!!secureTextEntry);
  return (
    <View style={{ gap: spacing.xs }}>
      <Text variant="label">{label}</Text>
      <View
        style={[
          styles.inputWrap,
          {
            backgroundColor: colors.surface,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border
          }
        ]}
      >
        <TextInput
          {...props}
          testID={testID}
          accessibilityLabel={label}
          multiline={multiline}
          secureTextEntry={hidden}
          onFocus={(e) => {
            setFocused(true);
            props.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            props.onBlur?.(e);
          }}
          placeholderTextColor={colors.textMuted}
          style={[
            styles.input,
            { color: colors.text },
            multiline ? { minHeight: 96, textAlignVertical: "top" } : null
          ]}
        />
        {secureTextEntry ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            onPress={() => setHidden((h) => !h)}
            hitSlop={8}
          >
            <Text variant="caption" color={colors.primary} style={{ fontWeight: "700" }}>
              {hidden ? "Afficher" : "Masquer"}
            </Text>
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text variant="caption" color={colors.danger} accessibilityRole="alert">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" muted>
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
  testID,
  accessibilityLabel
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
  accessibilityLabel?: string;
}) {
  const { colors } = useTheme();
  const body = (
    <View
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primarySoft : colors.surfaceAlt,
          borderColor: selected ? colors.primary : "transparent"
        }
      ]}
    >
      <RNText style={[sizes.label, { color: selected ? colors.primary : colors.text }]}>
        {label}
      </RNText>
    </View>
  );
  if (!onPress) return body;
  return (
    <Pressable
      testID={testID}
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      accessibilityState={{ selected: !!selected }}
    >
      {body}
    </Pressable>
  );
}

export function Row({
  children,
  style
}: {
  children: React.ReactNode;
  style?: StyleProp<ViewStyle>;
}) {
  return <View style={[{ flexDirection: "row", alignItems: "center" }, style]}>{children}</View>;
}

export function Card({
  children,
  style
}: {
  children: React.ReactNode;
  style?: StyleProp<ViewStyle>;
}) {
  const { colors } = useTheme();
  return (
    <View
      style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }, style]}
    >
      {children}
    </View>
  );
}

export function Screen({
  children,
  edges = ["top", "left", "right"],
  style
}: {
  children: React.ReactNode;
  edges?: Edge[];
  style?: StyleProp<ViewStyle>;
}) {
  const { colors } = useTheme();
  return (
    <SafeAreaView edges={edges} style={[{ flex: 1, backgroundColor: colors.background }, style]}>
      {children}
    </SafeAreaView>
  );
}

export function Loader({ label = fr.common.loading }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator size="large" color={colors.primary} />
    </View>
  );
}

export function EmptyState({
  title,
  body,
  glyph = "✨",
  action
}: {
  title: string;
  body?: string;
  glyph?: string;
  action?: React.ReactNode;
}) {
  return (
    <View style={styles.center}>
      <RNText style={{ fontSize: 44, marginBottom: spacing.md }}>{glyph}</RNText>
      <Text variant="heading" style={{ textAlign: "center" }}>
        {title}
      </Text>
      {body ? (
        <Text muted style={{ textAlign: "center", marginTop: spacing.sm, maxWidth: 320 }}>
          {body}
        </Text>
      ) : null}
      {action ? (
        <View style={{ marginTop: spacing.lg, alignSelf: "stretch" }}>{action}</View>
      ) : null}
    </View>
  );
}

export function ErrorState({ title, onRetry }: { title?: string; onRetry?: () => void }) {
  return (
    <EmptyState
      glyph="⚠️"
      title={title ?? fr.common.genericError}
      action={
        onRetry ? (
          <Button title={fr.common.retry} variant="secondary" onPress={onRetry} />
        ) : undefined
      }
    />
  );
}

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View
      style={[styles.badge, { backgroundColor: colors.primary }]}
      accessibilityLabel={`${count} non lus`}
    >
      <RNText style={{ color: colors.primaryText, fontSize: 11, fontWeight: "800" }}>
        {count > 99 ? "99+" : count}
      </RNText>
    </View>
  );
}

export function Divider() {
  const { colors } = useTheme();
  return <View style={{ height: StyleSheet.hairlineWidth, backgroundColor: colors.border }} />;
}

const styles = StyleSheet.create({
  button: {
    minHeight: 50,
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center"
  },
  iconButton: { alignItems: "center", justifyContent: "center", borderWidth: 1 },
  inputWrap: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1.5,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    minHeight: 50,
    gap: spacing.sm
  },
  input: { flex: 1, fontSize: 16, paddingVertical: spacing.md },
  chip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    minHeight: 36,
    justifyContent: "center"
  },
  card: { borderRadius: radius.lg, borderWidth: 1, padding: spacing.lg },
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    paddingHorizontal: 5,
    alignItems: "center",
    justifyContent: "center"
  }
});
