import React, { useEffect, useState } from "react";
import {
  ActivityIndicator,
  Image,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  type ImageStyle,
  type StyleProp,
  type TextInputProps,
  type ViewStyle
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { API_BASE_URL } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { fontSizes, radii, spacing, useTheme } from "../theme/theme";

/* ---------- Screen ---------- */

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom" | "left" | "right")[];
  contentStyle?: StyleProp<ViewStyle>;
};

export function Screen({ children, scroll = false, padded = true, edges = ["top", "bottom"], contentStyle }: ScreenProps) {
  const { colors } = useTheme();
  const inner = [padded && styles.padded, styles.grow, contentStyle];
  return (
    <SafeAreaView edges={edges} style={[styles.grow, { backgroundColor: colors.background }]}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.grow}>
        {scroll ? (
          <ScrollView contentContainerStyle={inner} keyboardShouldPersistTaps="handled">
            {children}
          </ScrollView>
        ) : (
          <View style={inner}>{children}</View>
        )}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

/* ---------- Text ---------- */

type AppTextProps = {
  children: React.ReactNode;
  variant?: "title" | "heading" | "body" | "caption" | "label";
  muted?: boolean;
  center?: boolean;
  color?: string;
  style?: StyleProp<ViewStyle>;
  numberOfLines?: number;
};

export function AppText({ children, variant = "body", muted, center, color, numberOfLines }: AppTextProps) {
  const { colors } = useTheme();
  const base = TEXT_STYLES[variant];
  return (
    <Text
      accessibilityRole={variant === "title" || variant === "heading" ? "header" : undefined}
      numberOfLines={numberOfLines}
      style={[base, { color: color ?? (muted ? colors.textMuted : colors.text) }, center && styles.center]}
    >
      {children}
    </Text>
  );
}

const TEXT_STYLES = StyleSheet.create({
  title: { fontSize: fontSizes.xxl, fontWeight: "800", letterSpacing: -0.5 },
  heading: { fontSize: fontSizes.lg, fontWeight: "700" },
  body: { fontSize: fontSizes.md, lineHeight: 22 },
  caption: { fontSize: fontSizes.sm, lineHeight: 18 },
  label: { fontSize: fontSizes.sm, fontWeight: "600" }
});

/* ---------- Button ---------- */

type ButtonProps = {
  label: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Button({ label, onPress, variant = "primary", loading, disabled, style, testID }: ButtonProps) {
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
      accessibilityRole="button"
      accessibilityState={{ disabled: !!inactive, busy: !!loading }}
      accessibilityLabel={label}
      disabled={inactive}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.button,
        { backgroundColor: palette.bg, borderColor: palette.border, opacity: inactive ? 0.5 : pressed ? 0.85 : 1 },
        style
      ]}
    >
      {loading ? <ActivityIndicator color={palette.fg} /> : <Text style={[styles.buttonLabel, { color: palette.fg }]}>{label}</Text>}
    </Pressable>
  );
}

/* ---------- TextField ---------- */

type FieldProps = TextInputProps & { label: string; error?: string | null; hint?: string };

export function TextField({ label, error, hint, style, ...rest }: FieldProps) {
  const { colors } = useTheme();
  const [focused, setFocused] = useState(false);
  return (
    <View style={styles.field}>
      <AppText variant="label">{label}</AppText>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        style={[
          styles.input,
          {
            backgroundColor: colors.surface,
            color: colors.text,
            borderColor: error ? colors.danger : focused ? colors.primary : colors.border
          },
          style
        ]}
        {...rest}
      />
      {error ? (
        <Text accessibilityLiveRegion="polite" style={[styles.fieldError, { color: colors.danger }]}>
          {error}
        </Text>
      ) : hint ? (
        <AppText variant="caption" muted>
          {hint}
        </AppText>
      ) : null}
    </View>
  );
}

/* ---------- Chip ---------- */

export function Chip({ label, selected, onPress, small }: { label: string; selected?: boolean; onPress?: () => void; small?: boolean }) {
  const { colors } = useTheme();
  const content = (
    <View
      style={[
        styles.chip,
        small && styles.chipSmall,
        {
          backgroundColor: selected ? colors.primary : colors.surface,
          borderColor: selected ? colors.primary : colors.border
        }
      ]}
    >
      <Text
        style={{ color: selected ? colors.primaryText : colors.text, fontSize: small ? fontSizes.xs : fontSizes.sm, fontWeight: "600" }}
      >
        {label}
      </Text>
    </View>
  );
  if (!onPress) return content;
  return (
    <Pressable accessibilityRole="button" accessibilityState={{ selected: !!selected }} accessibilityLabel={label} onPress={onPress}>
      {content}
    </Pressable>
  );
}

/* ---------- States ---------- */

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} style={styles.state}>
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? <AppText muted>{label}</AppText> : null}
    </View>
  );
}

export function EmptyState({ emoji, title, message, action }: { emoji: string; title: string; message: string; action?: React.ReactNode }) {
  return (
    <View style={styles.state}>
      <Text style={styles.emoji}>{emoji}</Text>
      <AppText variant="heading" center>
        {title}
      </AppText>
      <AppText muted center>
        {message}
      </AppText>
      {action}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      emoji="⚠️"
      title="Oups"
      message={message}
      action={onRetry ? <Button label="Réessayer" variant="secondary" onPress={onRetry} /> : undefined}
    />
  );
}

/* ---------- Photo ---------- */

// Browsers cannot attach an Authorization header to <img>, so on web the image
// is fetched with the token and displayed through an object URL (cached per id).
const webBlobUrls = new Map<string, string>();

function useWebBlobUrl(url: string | null | undefined, token: string | null): string | null {
  const [resolved, setResolved] = useState<string | null>(url ? (webBlobUrls.get(url) ?? null) : null);
  useEffect(() => {
    if (Platform.OS !== "web" || !url || !token) return;
    const cached = webBlobUrls.get(url);
    if (cached) {
      setResolved(cached);
      return;
    }
    let cancelled = false;
    fetch(`${API_BASE_URL}${url}`, { headers: { Authorization: `Bearer ${token}` } })
      .then((res) => (res.ok ? res.blob() : Promise.reject(new Error(String(res.status)))))
      .then((blob) => {
        const objectUrl = URL.createObjectURL(blob);
        webBlobUrls.set(url, objectUrl);
        if (!cancelled) setResolved(objectUrl);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [url, token]);
  return resolved;
}

export function useImageSource(url: string | null | undefined): { uri: string; headers?: Record<string, string> } | null {
  const { accessToken } = useAuth();
  const webUrl = useWebBlobUrl(url, accessToken);
  if (!url) return null;
  if (Platform.OS === "web") return webUrl ? { uri: webUrl } : null;
  return { uri: `${API_BASE_URL}${url}`, headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {} };
}

export function PhotoImage({ url, style, label }: { url: string | null | undefined; style: StyleProp<ImageStyle>; label?: string }) {
  const { colors } = useTheme();
  const source = useImageSource(url);
  const [failed, setFailed] = useState(false);
  if (!source || failed) {
    return <View accessibilityLabel={label} style={[style as StyleProp<ViewStyle>, { backgroundColor: colors.surfaceAlt }]} />;
  }
  return <Image accessibilityLabel={label} source={source} style={style} resizeMode="cover" onError={() => setFailed(true)} />;
}

export function Avatar({ url, name, size = 48 }: { url: string | null | undefined; name: string; size?: number }) {
  const { colors } = useTheme();
  const source = useImageSource(url);
  const dimension = { width: size, height: size, borderRadius: size / 2 };
  if (!source) {
    return (
      <View style={[dimension, styles.avatarFallback, { backgroundColor: colors.primarySoft }]}>
        <Text style={{ color: colors.primary, fontWeight: "800", fontSize: size * 0.4 }}>{name.slice(0, 1).toUpperCase()}</Text>
      </View>
    );
  }
  return <Image accessibilityLabel={`Photo de ${name}`} source={source} style={dimension} />;
}

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) return null;
  return (
    <View accessibilityLabel={`${count} non lus`} style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text style={{ color: colors.primaryText, fontSize: 11, fontWeight: "800" }}>{count > 99 ? "99+" : count}</Text>
    </View>
  );
}

/* ---------- Sheet (cross-platform action sheet / modal) ---------- */

export type SheetAction = { label: string; onPress: () => void; destructive?: boolean };

export function Sheet({
  visible,
  onClose,
  title,
  children
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}) {
  const { colors } = useTheme();
  return (
    <Modal animationType="slide" transparent visible={visible} onRequestClose={onClose}>
      <Pressable accessibilityLabel="Fermer" style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={onClose} />
      <View style={[styles.sheet, { backgroundColor: colors.surface }]}>
        <SafeAreaView edges={["bottom"]}>
          {title ? (
            <View style={styles.sheetTitle}>
              <AppText variant="heading">{title}</AppText>
            </View>
          ) : null}
          {children}
        </SafeAreaView>
      </View>
    </Modal>
  );
}

export function ActionSheet({
  visible,
  onClose,
  title,
  actions
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  actions: SheetAction[];
}) {
  const { colors } = useTheme();
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      {actions.map((action) => (
        <Pressable
          key={action.label}
          accessibilityRole="button"
          onPress={() => {
            onClose();
            action.onPress();
          }}
          style={({ pressed }) => [styles.sheetRow, { backgroundColor: pressed ? colors.surfaceAlt : "transparent" }]}
        >
          <Text style={{ color: action.destructive ? colors.danger : colors.text, fontSize: fontSizes.md, fontWeight: "600" }}>
            {action.label}
          </Text>
        </Pressable>
      ))}
      <Pressable accessibilityRole="button" onPress={onClose} style={styles.sheetRow}>
        <Text style={{ color: colors.textMuted, fontSize: fontSizes.md }}>Annuler</Text>
      </Pressable>
    </Sheet>
  );
}

export function Row({
  title,
  subtitle,
  onPress,
  right,
  danger
}: {
  title: string;
  subtitle?: string;
  onPress?: () => void;
  right?: React.ReactNode;
  danger?: boolean;
}) {
  const { colors } = useTheme();
  const body = (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <View style={styles.grow}>
        <Text style={{ color: danger ? colors.danger : colors.text, fontSize: fontSizes.md, fontWeight: "600" }}>{title}</Text>
        {subtitle ? (
          <AppText variant="caption" muted>
            {subtitle}
          </AppText>
        ) : null}
      </View>
      {right ?? (onPress ? <Text style={{ color: colors.textMuted, fontSize: fontSizes.lg }}>›</Text> : null)}
    </View>
  );
  return onPress ? (
    <Pressable accessibilityRole="button" accessibilityLabel={title} onPress={onPress}>
      {body}
    </Pressable>
  ) : (
    body
  );
}

const styles = StyleSheet.create({
  grow: { flex: 1 },
  padded: { padding: spacing.lg, gap: spacing.md },
  center: { textAlign: "center" },
  button: {
    minHeight: 50,
    borderRadius: radii.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: spacing.xl
  },
  buttonLabel: { fontSize: fontSizes.md, fontWeight: "700" },
  field: { gap: spacing.xs },
  input: {
    minHeight: 48,
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: Platform.OS === "ios" ? spacing.md : spacing.sm,
    fontSize: fontSizes.md
  },
  fieldError: { fontSize: fontSizes.sm },
  chip: { borderWidth: 1.5, borderRadius: radii.pill, paddingHorizontal: spacing.md, paddingVertical: spacing.sm },
  chipSmall: { paddingHorizontal: spacing.sm, paddingVertical: 4 },
  state: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  emoji: { fontSize: 48 },
  avatarFallback: { alignItems: "center", justifyContent: "center" },
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    paddingHorizontal: 5,
    alignItems: "center",
    justifyContent: "center"
  },
  backdrop: { flex: 1 },
  sheet: { borderTopLeftRadius: radii.lg, borderTopRightRadius: radii.lg, paddingTop: spacing.sm },
  sheetTitle: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  sheetRow: { paddingHorizontal: spacing.lg, paddingVertical: spacing.lg, alignItems: "flex-start" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: spacing.md
  }
});
