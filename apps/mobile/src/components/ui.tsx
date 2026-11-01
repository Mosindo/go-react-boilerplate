import { useState, type ReactNode } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text as RNText,
  TextInput,
  View,
  type ImageStyle,
  type StyleProp,
  type TextInputProps,
  type TextProps,
  type TextStyle,
  type ViewStyle,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { Image } from "expo-image";
import { photoSource } from "../api/client";
import { useAuth } from "../state/auth";
import { radius, spacing, useColors } from "../theme";
import { initials } from "../lib/format";

type Variant = "title" | "heading" | "body" | "caption" | "label";

const TEXT_STYLES: Record<Variant, TextStyle> = {
  title: { fontSize: 30, fontWeight: "700", letterSpacing: -0.5 },
  heading: { fontSize: 20, fontWeight: "700" },
  body: { fontSize: 16, lineHeight: 22 },
  caption: { fontSize: 13, lineHeight: 18 },
  label: { fontSize: 14, fontWeight: "600" },
};

export function Text({ variant = "body", muted, style, ...props }: TextProps & { variant?: Variant; muted?: boolean }) {
  const colors = useColors();
  return <RNText {...props} style={[TEXT_STYLES[variant], { color: muted ? colors.textMuted : colors.text }, style]} />;
}

export function Screen({
  children,
  scroll,
  padded = true,
  edges = ["top", "bottom"],
  style,
}: {
  children: ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: ("top" | "bottom" | "left" | "right")[];
  style?: StyleProp<ViewStyle>;
}) {
  const colors = useColors();
  const content = padded ? { padding: spacing.lg } : undefined;
  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: colors.background }]}>
      {scroll ? (
        <ScrollView
          contentContainerStyle={[content, styles.grow, style]}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={[styles.flex, content, style]}>{children}</View>
      )}
    </SafeAreaView>
  );
}

export function Button({
  title,
  onPress,
  variant = "primary",
  loading,
  disabled,
  icon,
  style,
  testID,
}: {
  title: string;
  onPress: () => void;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  loading?: boolean;
  disabled?: boolean;
  icon?: keyof typeof Ionicons.glyphMap;
  style?: StyleProp<ViewStyle>;
  testID?: string;
}) {
  const colors = useColors();
  const inactive = disabled || loading;
  const palette = {
    primary: { bg: colors.primary, fg: colors.onPrimary, border: colors.primary },
    secondary: { bg: colors.surface, fg: colors.text, border: colors.border },
    ghost: { bg: "transparent", fg: colors.primary, border: "transparent" },
    danger: { bg: colors.danger, fg: "#FFFFFF", border: colors.danger },
  }[variant];
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
        { backgroundColor: palette.bg, borderColor: palette.border, opacity: inactive ? 0.55 : pressed ? 0.85 : 1 },
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={palette.fg} />
      ) : (
        <>
          {icon ? <Ionicons color={palette.fg} name={icon} size={18} style={{ marginRight: spacing.sm }} /> : null}
          <RNText style={[TEXT_STYLES.label, { color: palette.fg, fontSize: 16 }]}>{title}</RNText>
        </>
      )}
    </Pressable>
  );
}

export function TextField({
  label,
  error,
  secure,
  style,
  ...props
}: TextInputProps & { label: string; error?: string | null; secure?: boolean }) {
  const colors = useColors();
  const [hidden, setHidden] = useState(!!secure);
  return (
    <View style={{ marginBottom: spacing.md }}>
      <Text muted style={{ marginBottom: spacing.xs }} variant="label">
        {label}
      </Text>
      <View
        style={[styles.field, { backgroundColor: colors.surface, borderColor: error ? colors.danger : colors.border }]}
      >
        <TextInput
          accessibilityLabel={label}
          placeholderTextColor={colors.textMuted}
          secureTextEntry={hidden}
          style={[styles.input, { color: colors.text }, style]}
          {...props}
        />
        {secure ? (
          <Pressable
            accessibilityLabel={hidden ? "Afficher le mot de passe" : "Masquer le mot de passe"}
            hitSlop={8}
            onPress={() => setHidden((h) => !h)}
          >
            <Ionicons color={colors.textMuted} name={hidden ? "eye-outline" : "eye-off-outline"} size={20} />
          </Pressable>
        ) : null}
      </View>
      {error ? (
        <Text
          accessibilityLiveRegion="polite"
          style={{ color: colors.danger, marginTop: spacing.xs }}
          variant="caption"
        >
          {error}
        </Text>
      ) : null}
    </View>
  );
}

export function Chip({ label, selected, onPress }: { label: string; selected?: boolean; onPress?: () => void }) {
  const colors = useColors();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: !!selected }}
      disabled={!onPress}
      onPress={onPress}
      style={[
        styles.chip,
        {
          backgroundColor: selected ? colors.primary : colors.surface,
          borderColor: selected ? colors.primary : colors.border,
        },
      ]}
    >
      <RNText style={[TEXT_STYLES.label, { color: selected ? colors.onPrimary : colors.text }]}>{label}</RNText>
    </Pressable>
  );
}

/** Photo authentifiee (cache disque via expo-image). */
export function PhotoImage({
  photoId,
  style,
  contentFit = "cover",
}: {
  photoId: string;
  style?: StyleProp<ImageStyle>;
  contentFit?: "cover" | "contain";
}) {
  const { accessToken } = useAuth();
  const colors = useColors();
  return (
    <Image
      cachePolicy="disk"
      contentFit={contentFit}
      source={photoSource(photoId, accessToken)}
      style={[{ backgroundColor: colors.surfaceAlt }, style]}
      transition={150}
    />
  );
}

export function Avatar({ photoId, name, size = 56 }: { photoId: string | null; name: string; size?: number }) {
  const colors = useColors();
  const dims = { width: size, height: size, borderRadius: size / 2 };
  if (photoId) {
    return <PhotoImage photoId={photoId} style={dims} />;
  }
  return (
    <View style={[dims, styles.center, { backgroundColor: colors.surfaceAlt }]}>
      <RNText style={[TEXT_STYLES.heading, { color: colors.textMuted }]}>{initials(name)}</RNText>
    </View>
  );
}

export function Loading({ label }: { label?: string }) {
  const colors = useColors();
  return (
    <View accessibilityLabel={label ?? "Chargement"} style={[styles.flex, styles.center]}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text muted style={{ marginTop: spacing.md }}>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

export function EmptyState({
  icon,
  title,
  message,
  action,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  action?: ReactNode;
}) {
  const colors = useColors();
  return (
    <View style={[styles.flex, styles.center, { padding: spacing.xl }]}>
      <Ionicons color={colors.textMuted} name={icon} size={48} />
      <Text style={{ marginTop: spacing.md, textAlign: "center" }} variant="heading">
        {title}
      </Text>
      {message ? (
        <Text muted style={{ marginTop: spacing.sm, textAlign: "center" }}>
          {message}
        </Text>
      ) : null}
      {action ? <View style={{ marginTop: spacing.lg }}>{action}</View> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      action={onRetry ? <Button onPress={onRetry} title="Réessayer" variant="secondary" /> : undefined}
      icon="cloud-offline-outline"
      message={message}
      title="Oups"
    />
  );
}

export function Row({
  icon,
  title,
  subtitle,
  onPress,
  danger,
  right,
}: {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  subtitle?: string;
  onPress?: () => void;
  danger?: boolean;
  right?: ReactNode;
}) {
  const colors = useColors();
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.8 : 1 },
      ]}
    >
      {icon ? (
        <Ionicons
          color={danger ? colors.danger : colors.primary}
          name={icon}
          size={22}
          style={{ marginRight: spacing.md }}
        />
      ) : null}
      <View style={styles.flex}>
        <Text style={danger ? { color: colors.danger } : undefined} variant="label">
          {title}
        </Text>
        {subtitle ? (
          <Text muted variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ?? (onPress ? <Ionicons color={colors.textMuted} name="chevron-forward" size={18} /> : null)}
    </Pressable>
  );
}

export function Badge({ count }: { count: number }) {
  const colors = useColors();
  if (count <= 0) {
    return null;
  }
  return (
    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
      <RNText style={{ color: colors.onPrimary, fontSize: 11, fontWeight: "700" }}>{count > 99 ? "99+" : count}</RNText>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  grow: { flexGrow: 1 },
  center: { alignItems: "center", justifyContent: "center" },
  button: {
    minHeight: 50,
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.xl,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
  },
  field: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
  },
  input: { flex: 1, minHeight: 48, fontSize: 16 },
  chip: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm + 2,
    borderRadius: radius.pill,
    borderWidth: 1,
    marginRight: spacing.sm,
    marginBottom: spacing.sm,
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    padding: spacing.lg,
    borderRadius: radius.md,
    borderWidth: 1,
    marginBottom: spacing.sm,
  },
  badge: {
    minWidth: 18,
    height: 18,
    borderRadius: 9,
    paddingHorizontal: 5,
    alignItems: "center",
    justifyContent: "center",
  },
});
