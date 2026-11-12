import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { spacing, useTheme } from "../theme";
import { IconButton } from "./Button";
import { Text } from "./Text";

export function Screen({
  children,
  scroll = false,
  edges = ["top"],
  padded = true,
  keyboard = false,
  contentStyle
}: {
  children: React.ReactNode;
  scroll?: boolean;
  edges?: Edge[];
  padded?: boolean;
  keyboard?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
}) {
  const { colors } = useTheme();
  const inner = scroll ? (
    <ScrollView
      keyboardShouldPersistTaps="handled"
      contentContainerStyle={[padded && styles.padded, styles.scrollContent, contentStyle]}
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.fill, padded && styles.padded, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={[styles.fill, { backgroundColor: colors.background }]}>
      {keyboard ? (
        <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
          {inner}
        </KeyboardAvoidingView>
      ) : (
        inner
      )}
    </SafeAreaView>
  );
}

export function Header({ title, onBack, right, subtitle }: { title: string; subtitle?: string; onBack?: () => void; right?: React.ReactNode }) {
  return (
    <View style={styles.header}>
      {onBack ? <IconButton icon="chevron-back" label="Retour" onPress={onBack} /> : null}
      <View style={styles.headerText}>
        <Text variant="heading" numberOfLines={1} accessibilityRole="header">
          {title}
        </Text>
        {subtitle ? (
          <Text variant="caption" tone="muted" numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
}

export function Section({ title, children, footer }: { title?: string; children: React.ReactNode; footer?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.section}>
      {title ? (
        <Text variant="overline" tone="subtle" style={styles.sectionTitle}>
          {title}
        </Text>
      ) : null}
      <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>{children}</View>
      {footer ? (
        <Text variant="caption" tone="subtle" style={styles.sectionFooter}>
          {footer}
        </Text>
      ) : null}
    </View>
  );
}

export function ProgressBar({ value }: { value: number }) {
  const { colors } = useTheme();
  return (
    <View
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}
      style={[styles.track, { backgroundColor: colors.surfaceMuted }]}
    >
      <View style={[styles.bar, { width: `${Math.max(4, Math.min(100, value * 100))}%`, backgroundColor: colors.primary }]} />
    </View>
  );
}

export function Badge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count <= 0) {
    return null;
  }
  return (
    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
      <Text variant="caption" style={{ color: colors.onPrimary, fontWeight: "700", fontSize: 11 }}>
        {count > 99 ? "99+" : count}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  padded: { paddingHorizontal: spacing.lg },
  scrollContent: { paddingBottom: spacing.xxxl },
  header: { flexDirection: "row", alignItems: "center", gap: spacing.xs, minHeight: 56, paddingHorizontal: spacing.xs },
  headerText: { flex: 1 },
  section: { marginTop: spacing.lg, gap: spacing.xs },
  sectionTitle: { paddingHorizontal: spacing.xxs },
  sectionFooter: { paddingHorizontal: spacing.xxs },
  card: { borderRadius: 18, borderWidth: 1, overflow: "hidden" },
  track: { height: 6, borderRadius: 3, overflow: "hidden" },
  bar: { height: 6, borderRadius: 3 },
  badge: { minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 5, alignItems: "center", justifyContent: "center" }
});
