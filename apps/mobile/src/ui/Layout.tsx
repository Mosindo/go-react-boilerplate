import React from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { SafeAreaView, type Edge } from "react-native-safe-area-context";
import { useTheme } from "../theme/ThemeProvider";
import { spacing } from "../theme/palette";
import { IconButton } from "./Button";
import { Text } from "./Text";

type ScreenProps = {
  children: React.ReactNode;
  scroll?: boolean;
  padded?: boolean;
  edges?: Edge[];
  keyboard?: boolean;
  contentStyle?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Screen({ children, scroll, padded = true, edges = ["top"], keyboard, contentStyle, testID }: ScreenProps) {
  const { colors } = useTheme();
  const inner = scroll ? (
    <ScrollView
      contentContainerStyle={[padded ? styles.padded : null, styles.scrollContent, contentStyle]}
      keyboardShouldPersistTaps="handled"
      showsVerticalScrollIndicator={false}
    >
      {children}
    </ScrollView>
  ) : (
    <View style={[styles.flex, padded ? styles.padded : null, contentStyle]}>{children}</View>
  );
  return (
    <SafeAreaView edges={edges} style={[styles.flex, { backgroundColor: colors.background }]} testID={testID}>
      {keyboard ? (
        <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
          {inner}
        </KeyboardAvoidingView>
      ) : (
        inner
      )}
    </SafeAreaView>
  );
}

type HeaderProps = {
  title?: string;
  onBack?: () => void;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, onBack, right, large }: HeaderProps) {
  return (
    <View style={[styles.header, large ? styles.headerLarge : null]}>
      {onBack ? <IconButton icon="chevron-back" accessibilityLabel="Retour" onPress={onBack} /> : null}
      <View style={styles.flex}>
        {title ? (
          <Text variant={large ? "title" : "heading"} accessibilityRole="header" numberOfLines={1}>
            {title}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
}

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const { colors } = useTheme();
  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }, style]}>{children}</View>
  );
}

export function Section({ title, children, action }: { title: string; children: React.ReactNode; action?: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeader}>
        <Text variant="overline" tone="subtle">
          {title.toUpperCase()}
        </Text>
        {action}
      </View>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  padded: { paddingHorizontal: spacing.lg },
  scrollContent: { paddingBottom: spacing.xxxl, flexGrow: 1 },
  header: { flexDirection: "row", alignItems: "center", gap: spacing.xs, minHeight: 52, paddingVertical: spacing.xxs },
  headerLarge: { minHeight: 64 },
  card: { borderRadius: 20, borderWidth: StyleSheet.hairlineWidth, padding: spacing.md },
  section: { gap: spacing.sm, marginTop: spacing.lg },
  sectionHeader: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" }
});
