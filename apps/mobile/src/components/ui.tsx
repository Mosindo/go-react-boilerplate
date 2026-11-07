import React from "react";
import {
  ActivityIndicator,
  Image,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  type StyleProp,
  type TextInputProps,
  type ViewStyle,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { font, radius, spacing, useTheme } from "../theme/theme";
import { mediaUrl } from "../api/config";

export function Screen({
  children,
  scroll = false,
  padded = true,
  edges = ["top", "bottom"],
}: {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom")[];
}) {
  const { colors } = useTheme();
  const pad = padded ? { paddingHorizontal: spacing.lg } : null;
  return (
    <SafeAreaView edges={edges} style={{ flex: 1, backgroundColor: colors.bg }}>
      {scroll ? (
        <ScrollView
          contentContainerStyle={[{ paddingVertical: spacing.lg, gap: spacing.lg }, pad]}
          keyboardShouldPersistTaps="handled"
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[{ flex: 1 }, pad]}>{children}</View>
      )}
    </SafeAreaView>
  );
}

export function AppText({
  children,
  variant = "body",
  style,
  ...rest
}: {
  children: React.ReactNode;
  variant?: "title" | "heading" | "body" | "small" | "muted" | "error";
  style?: StyleProp<import("react-native").TextStyle>;
} & Omit<import("react-native").TextProps, "style">) {
  const { colors } = useTheme();
  const styles = {
    title: { fontSize: font.xl, fontWeight: "800" as const, color: colors.text, letterSpacing: -0.5 },
    heading: { fontSize: font.lg, fontWeight: "700" as const, color: colors.text },
    body: { fontSize: font.md, color: colors.text, lineHeight: 22 },
    small: { fontSize: font.sm, color: colors.text },
    muted: { fontSize: font.sm, color: colors.textMuted, lineHeight: 20 },
    error: { fontSize: font.sm, color: colors.danger },
  };
  return (
    <Text style={[styles[variant], style]} {...rest}>
      {children}
    </Text>
  );
}

export function Button({
  title,
  onPress,
  variant = "primary",
  loading = false,
  disabled = false,
  testID,
  style,
}: {
  title: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
}) {
  const { colors } = useTheme();
  const palette = {
    primary: { bg: colors.primary, fg: colors.primaryText, border: colors.primary },
    secondary: { bg: colors.surface, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    danger: { bg: colors.danger, fg: "#fff", border: colors.danger },
  }[variant];
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: inactive, busy: loading }}
      accessibilityLabel={title}
      testID={testID}
      onPress={inactive ? undefined : onPress}
      style={({ pressed }) => [
        styles.button,
        {
          backgroundColor: palette.bg,
          borderColor: palette.border,
          opacity: inactive ? 0.55 : pressed ? 0.85 : 1,
        },
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <Text style={{ color: palette.fg, fontSize: font.md, fontWeight: "700" }}>{title}</Text>
      )}
    </Pressable>
  );
}

export function TextField({
  label,
  error,
  hint,
  ...input
}: { label: string; error?: string | null; hint?: string } & TextInputProps) {
  const { colors } = useTheme();
  return (
    <View style={{ gap: spacing.xs }}>
      <AppText variant="small" style={{ fontWeight: "600" }}>
        {label}
      </AppText>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        {...input}
        style={[
          styles.input,
          {
            borderColor: error ? colors.danger : colors.border,
            backgroundColor: colors.surface,
            color: colors.text,
          },
          input.multiline ? { minHeight: 96, textAlignVertical: "top" } : null,
        ]}
      />
      {error ? (
        <AppText variant="error">{error}</AppText>
      ) : hint ? (
        <AppText variant="muted">{hint}</AppText>
      ) : null}
    </View>
  );
}

export function Chip({
  label,
  selected = false,
  onPress,
  testID,
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole={onPress ? "button" : "text"}
      accessibilityState={{ selected }}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primary : colors.surfaceAlt,
          borderColor: selected ? colors.primary : colors.border,
        },
      ]}
    >
      <Text
        style={{ color: selected ? colors.primaryText : colors.text, fontSize: font.sm, fontWeight: "600" }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

export function Avatar({ uri, name, size = 48 }: { uri?: string | null; name: string; size?: number }) {
  const { colors } = useTheme();
  const url = mediaUrl(uri);
  return url ? (
    <Image
      accessibilityLabel={`Photo de ${name}`}
      source={{ uri: url }}
      style={{ width: size, height: size, borderRadius: size / 2, backgroundColor: colors.surfaceAlt }}
    />
  ) : (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        backgroundColor: colors.accent,
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Text style={{ color: colors.bg, fontWeight: "800", fontSize: size / 2.4 }}>
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? <AppText variant="muted">{label}</AppText> : null}
    </View>
  );
}

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction,
}: {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View style={styles.center}>
      <AppText variant="heading" style={{ textAlign: "center" }}>
        {title}
      </AppText>
      <AppText variant="muted" style={{ textAlign: "center", maxWidth: 300 }}>
        {message}
      </AppText>
      {actionLabel && onAction ? <Button title={actionLabel} variant="secondary" onPress={onAction} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityRole="alert">
      <AppText variant="heading" style={{ textAlign: "center" }}>
        Oups
      </AppText>
      <AppText variant="muted" style={{ textAlign: "center", maxWidth: 300 }}>
        {message}
      </AppText>
      {onRetry ? <Button title="Réessayer" variant="secondary" onPress={onRetry} /> : null}
    </View>
  );
}

export function Banner({ message, tone = "error" }: { message: string; tone?: "error" | "info" }) {
  const { colors } = useTheme();
  return (
    <View
      accessibilityRole="alert"
      style={{
        padding: spacing.md,
        borderRadius: radius.md,
        backgroundColor: tone === "error" ? colors.surfaceAlt : colors.surfaceAlt,
        borderWidth: 1,
        borderColor: tone === "error" ? colors.danger : colors.border,
      }}
    >
      <AppText variant={tone === "error" ? "error" : "muted"}>{message}</AppText>
    </View>
  );
}

export function Row({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  return (
    <View style={[{ flexDirection: "row", alignItems: "center", gap: spacing.md }, style]}>{children}</View>
  );
}

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        {
          backgroundColor: colors.surface,
          borderRadius: radius.lg,
          borderWidth: 1,
          borderColor: colors.border,
          padding: spacing.lg,
          gap: spacing.md,
        },
        style,
      ]}
    >
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  button: {
    minHeight: 50,
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    alignItems: "center",
    justifyContent: "center",
  },
  input: {
    minHeight: 50,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    fontSize: font.md,
  },
  chip: {
    borderWidth: 1,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
  },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
});
