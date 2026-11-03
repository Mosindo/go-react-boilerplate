import React from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
  type StyleProp,
  type TextInputProps,
  type ViewStyle
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { radius, spacing, useTheme } from "../theme";

type ButtonProps = {
  title: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function Button({ title, onPress, variant = "primary", loading, disabled, testID, style }: ButtonProps) {
  const theme = useTheme();
  const palette = {
    primary: { bg: theme.primary, fg: theme.primaryText, border: theme.primary },
    secondary: { bg: theme.surface, fg: theme.text, border: theme.border },
    ghost: { bg: "transparent", fg: theme.primary, border: "transparent" },
    danger: { bg: theme.surface, fg: theme.danger, border: theme.danger }
  }[variant];
  const inactive = disabled || loading;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={title}
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.button,
        { backgroundColor: palette.bg, borderColor: palette.border, opacity: inactive ? 0.5 : pressed ? 0.85 : 1 },
        style
      ]}
    >
      {loading ? <ActivityIndicator color={palette.fg} /> : <Text style={[styles.buttonText, { color: palette.fg }]}>{title}</Text>}
    </Pressable>
  );
}

type FieldProps = TextInputProps & { label: string; error?: string | null; hint?: string };

export function TextField({ label, error, hint, style, ...input }: FieldProps) {
  const theme = useTheme();
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: theme.text }]}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={theme.textMuted}
        {...input}
        style={[
          styles.input,
          { color: theme.text, backgroundColor: theme.surface, borderColor: error ? theme.danger : theme.border },
          input.multiline && styles.multiline,
          style
        ]}
      />
      {error ? (
        <Text accessibilityRole="alert" style={[styles.hint, { color: theme.danger }]}>
          {error}
        </Text>
      ) : hint ? (
        <Text style={[styles.hint, { color: theme.textMuted }]}>{hint}</Text>
      ) : null}
    </View>
  );
}

export function Chip({ label, selected, onPress, testID }: { label: string; selected?: boolean; onPress?: () => void; testID?: string }) {
  const theme = useTheme();
  const body = (
    <View
      style={[
        styles.chip,
        { backgroundColor: selected ? theme.primary : theme.surfaceAlt, borderColor: selected ? theme.primary : theme.border }
      ]}
    >
      <Text style={{ color: selected ? theme.primaryText : theme.text, fontSize: 14, fontWeight: "600" }}>{label}</Text>
    </View>
  );
  if (!onPress) {
    return body;
  }
  return (
    <Pressable accessibilityRole="button" accessibilityState={{ selected: !!selected }} onPress={onPress} testID={testID}>
      {body}
    </Pressable>
  );
}

export function ChipRow({ children }: { children: React.ReactNode }) {
  return <View style={styles.chipRow}>{children}</View>;
}

export function Screen({ children, scroll = true, padded = true }: { children: React.ReactNode; scroll?: boolean; padded?: boolean }) {
  const theme = useTheme();
  const content = padded ? styles.padded : undefined;
  return (
    <SafeAreaView edges={["bottom"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      {scroll ? (
        <ScrollView contentContainerStyle={[content, styles.grow]} keyboardShouldPersistTaps="handled">
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.flex, content]}>{children}</View>
      )}
    </SafeAreaView>
  );
}

export function Title({ children }: { children: React.ReactNode }) {
  const theme = useTheme();
  return (
    <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
      {children}
    </Text>
  );
}

export function Body({ children, muted, center }: { children: React.ReactNode; muted?: boolean; center?: boolean }) {
  const theme = useTheme();
  return <Text style={[styles.body, { color: muted ? theme.textMuted : theme.text }, center && { textAlign: "center" }]}>{children}</Text>;
}

export function Loading({ label }: { label?: string }) {
  const theme = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} style={styles.center}>
      <ActivityIndicator color={theme.primary} size="large" />
      {label ? <Text style={[styles.body, { color: theme.textMuted, marginTop: spacing.md }]}>{label}</Text> : null}
    </View>
  );
}

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction
}: {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View style={styles.center}>
      <Title>{title}</Title>
      <Body muted center>
        {message}
      </Body>
      {actionLabel && onAction ? (
        <Button title={actionLabel} onPress={onAction} variant="secondary" style={{ marginTop: spacing.lg }} />
      ) : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return <EmptyState title="Oups" message={message} actionLabel={onRetry ? "Réessayer" : undefined} onAction={onRetry} />;
}

export function ErrorText({ message }: { message: string | null }) {
  const theme = useTheme();
  if (!message) {
    return null;
  }
  return (
    <Text accessibilityRole="alert" style={[styles.body, { color: theme.danger, marginBottom: spacing.md }]}>
      {message}
    </Text>
  );
}

export function ToggleRow({
  label,
  description,
  value,
  onChange
}: {
  label: string;
  description?: string;
  value: boolean;
  onChange: (next: boolean) => void;
}) {
  const theme = useTheme();
  return (
    <View style={styles.toggleRow}>
      <View style={styles.flex}>
        <Text style={[styles.label, { color: theme.text }]}>{label}</Text>
        {description ? <Text style={[styles.hint, { color: theme.textMuted }]}>{description}</Text> : null}
      </View>
      <Switch accessibilityLabel={label} value={value} onValueChange={onChange} trackColor={{ true: theme.primary, false: theme.border }} />
    </View>
  );
}

export function Badge({ count }: { count: number }) {
  const theme = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View style={[styles.badge, { backgroundColor: theme.primary }]}>
      <Text style={{ color: theme.primaryText, fontSize: 12, fontWeight: "700" }}>{count > 99 ? "99+" : count}</Text>
    </View>
  );
}

export const layout = StyleSheet.create({ row: { flexDirection: "row", alignItems: "center", gap: spacing.md } });

const styles = StyleSheet.create({
  flex: { flex: 1 },
  grow: { flexGrow: 1 },
  padded: { padding: spacing.lg, gap: spacing.md },
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl, gap: spacing.sm },
  button: {
    minHeight: 48,
    borderRadius: radius.pill,
    borderWidth: 1.5,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.xl
  },
  buttonText: { fontSize: 16, fontWeight: "700" },
  field: { gap: spacing.xs },
  label: { fontSize: 15, fontWeight: "600" },
  input: {
    minHeight: 48,
    borderWidth: 1.5,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    fontSize: 16
  },
  multiline: { minHeight: 96, textAlignVertical: "top" },
  hint: { fontSize: 13 },
  chip: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radius.pill, borderWidth: 1 },
  chipRow: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  title: { fontSize: 26, fontWeight: "800", letterSpacing: -0.3 },
  body: { fontSize: 16, lineHeight: 22 },
  toggleRow: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.sm },
  badge: { minWidth: 22, height: 22, borderRadius: 11, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 }
});
